package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode"

	"service-wilma-accounts/config"

	"github.com/dimchansky/utfbom"
	mssqlutils "github.com/pasiol/go-mssql-utils"
	pq "github.com/pasiol/gopq"

	_ "github.com/denisenkom/go-mssqldb"
)

type WilmaUserSQL struct {
	Email sql.NullString
}

var (
	newAccountConfig     = ""
	linkOldAccountConfig = ""
)

func insertStudent(db *sql.DB, u config.WilmaUser) error {
	ctx := context.Background()
	err := db.PingContext(ctx)
	if err != nil {
		log.Fatal(err.Error())
	}
	var archieved int
	if u.Archieved {
		archieved = 1
	} else {
		archieved = 0
	}
	tsql := fmt.Sprintf("INSERT INTO %s ([id], [user type], [nickname],[first names], [last name], [personal email], [phone number], [personal id], [student id], [email], [archieved]) VALUES(%s, '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', '%s', %d);", table, u.ID, u.UserType, u.NickName, u.FirstNames, u.LastName, u.PersonalEmail, u.PhoneNumber, u.PersonalID, u.StudentID, u.Email, archieved)
	c, err := db.QueryContext(ctx, tsql)
	if err != nil {
		return err
	}
	defer c.Close()
	return nil
}

func personLookup(db *sql.DB, u config.WilmaUser) (string, error) {
	ctx := context.Background()
	err := db.PingContext(ctx)
	if err != nil {
		return "", err
	}
	tsql := fmt.Sprintf("SELECT [email] FROM %s WHERE [personal id]='%s';", table, u.PersonalID)

	rows, err := db.QueryContext(ctx, tsql)
	if err != nil {
		return "", err
	}

	Email := ""
	var user WilmaUserSQL
	for rows.Next() {
		err = rows.Scan(&user.Email)
		log.Printf("lookup: %s", user.Email.String)
		if err != nil {
			return "", err
		} else {
			break
		}
	}

	if strings.Contains(user.Email.String, "@some.domain.com") {
		b := strings.Replace(user.Email.String, "@some.domain.com", "", 1)
		reg, err := regexp.Compile("[0-9]")
		if err != nil {
			log.Fatal(err)
		}
		r := reg.ReplaceAllString(b, "")
		if len(r) > 0 {
			Email = user.Email.String
		}
	}
	return Email, nil
}

func accountLookup(db *sql.DB, email string) (string, error) {

	ctx := context.Background()
	err := db.PingContext(ctx)
	if err != nil {
		return "", err
	}
	tsql := fmt.Sprintf("SELECT [email] FROM %s WHERE [email]='%s';", table, email)
	rows, err := db.QueryContext(ctx, tsql)
	if err != nil {
		return "", err
	}
	var user WilmaUserSQL
	for rows.Next() {
		err := rows.Scan(&user.Email)
		if err != nil {
			return "", err
		}
	}

	return user.Email.String, nil
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsGraphic(r) {
			return r
		}
		return -1
	}, s)
}

func emailNameSanitizer(n string) string {
	sanitizedName := strings.ToLower(n)
	sanitizedName = strings.ReplaceAll(sanitizedName, "ä", "a")
	sanitizedName = strings.ReplaceAll(sanitizedName, "à", "a")
	sanitizedName = strings.ReplaceAll(sanitizedName, "ö", "o")
	sanitizedName = strings.ReplaceAll(sanitizedName, "å", "a")
	sanitizedName = strings.ReplaceAll(sanitizedName, "è", "e")
	sanitizedName = strings.ReplaceAll(sanitizedName, "é", "e")
	sanitizedName = strings.ReplaceAll(sanitizedName, "ć", "c")
	sanitizedName = strings.ReplaceAll(sanitizedName, " ", ".")

	reg, err := regexp.Compile("[^a-zA-Z0-9.-]+")
	if err != nil {
		log.Fatal(err)
	}
	sanitizedName = reg.ReplaceAllString(sanitizedName, "")

	return sanitizedName
}

func initialsInEmail(u config.WilmaUser) string {
	nickname := strings.ToLower(u.NickName)
	firstNames := strings.ToLower(u.FirstNames)
	firstNames = strings.Replace(firstNames, nickname, "", 1)
	initials := ""
	if firstNames == "" {
		return u.Email
	}

	for _, name := range strings.Split(firstNames, " ") {
		if len(name) > 0 {
			initial := []byte(name)[0]
			initials = initials + string(initial) + "."
		}
	}

	if len(initials) > 1 {
		return emailNameSanitizer(nickname+"."+initials[0:len(initials)-1]+"."+strings.ToLower(u.LastName)) + "@some.domain.com"
	}
	return emailNameSanitizer(nickname+"."+strings.ToLower(u.LastName)) + "@some.domain.com"
}

func allNamesInEmail(u config.WilmaUser) string {
	nickname := strings.ToLower(u.NickName)
	firstNames := strings.ToLower(u.FirstNames)
	firstNames = strings.Replace(firstNames+" ", nickname+" ", "", 1)
	if firstNames == "" {
		return u.Email
	}
	return emailNameSanitizer(nickname+"."+firstNames) + "@some.domain.com"
}

func counterInEmail(u config.WilmaUser, c int) string {
	return emailNameSanitizer(u.NickName+"."+strconv.Itoa(c)+"."+u.LastName) + "@some.domain.com"
}

func createNewWilmaAccountSQL(db *sql.DB, user config.WilmaUser) error {
	err := insertStudent(db, user)
	if err != nil {
		return err
	}
	return nil
}

func linkNewWilmaAccountToStudent(student config.WilmaUser) error {
	time.Sleep(3 * time.Second)
	studentLinkToWilmaUserAccountFile, err := config.UpdateStudentXML(student)
	if err != nil {
		return err
	}
	c := config.GetPrimusConfig()
	_, errorCount, err := pq.ExecuteAtomicImportQuery(studentLinkToWilmaUserAccountFile, c.PrimusHost, c.PrimusPort, c.PrimusUser, c.PrimusPassword, linkOldAccountConfig)
	if err != nil && errorCount > 0 {
		return errors.New("linking new wilma user account failed")
	}

	return nil
}

func createNewWilmaAccountPrimus(user config.WilmaUser) error {
	primusWilmaUserFile, err := config.UserAccountXML(user)
	if err != nil {
		return err
	}
	c := config.GetPrimusConfig()
	_, errorCount, err := pq.ExecuteAtomicImportQuery(primusWilmaUserFile, c.PrimusHost, c.PrimusPort, c.PrimusUser, c.PrimusPassword, newAccountConfig)
	if err != nil && errorCount > 0 {
		return errors.New("creating new wilma user account failed")
	}

	return nil
}

func newWilmaAccount(db *sql.DB, user config.WilmaUser) (string, error) {
	accountEmail, err := accountLookup(db, user.Email)
	if err != nil {
		return "", err
	}
	if accountEmail == "" {
		err := createNewWilmaAccountSQL(db, user)
		if err != nil {
			return "", err
		}
	} else { // trying initials
		email := initialsInEmail(user)
		accountEmail, err := accountLookup(db, email)
		if err != nil {
			return "", err
		}
		if accountEmail == "" {
			user.Email = email
			err := createNewWilmaAccountSQL(db, user)
			if err != nil {
				return "", err
			}
		} else { // trying all names
			email := allNamesInEmail(user)
			accountEmail, err := accountLookup(db, email)
			if err != nil {
				return "", err
			}
			if accountEmail == "" {
				user.Email = email
				err := createNewWilmaAccountSQL(db, user)
				if err != nil {
					return "", err
				}
			} else { // trying numbers
				count := 1
				for {
					email := counterInEmail(user, count)
					accountEmail, err := accountLookup(db, email)
					if err != nil {
						return "", err
					}
					if accountEmail == "" {
						user.Email = email
						err := createNewWilmaAccountSQL(db, user)
						if err != nil {
							return "", err
						}
						break
					}
					count = count + 1
				}
			}
		}
	}
	if len(user.Email) < 19 {
		return "", errors.New("missing data, too short email")
	}
	return user.Email, nil
}

func readCSVFromString(data string) ([][]string, error) {

	r := csv.NewReader(utfbom.SkipOnly(strings.NewReader(data)))
	r.Comma = ';'
	lines, err := r.ReadAll()
	if err != nil {
		return [][]string{}, err
	}

	return lines, nil
}

func mapWilmaUsers(data []string) config.WilmaUser {
	u := config.WilmaUser{}

	u.ID = data[0]
	u.UserType = data[1]
	u.NickName = strings.TrimRight(strings.TrimLeft((data[2]), " "), " ")
	u.FirstNames = strings.TrimRight(strings.TrimLeft((data[3]), " "), " ")
	u.LastName = strings.TrimRight(strings.TrimLeft((data[4]), " "), " ")
	u.PersonalEmail = clean(data[5])
	u.PhoneNumber = clean(data[6])
	u.PersonalID = clean(data[7])
	u.StudentID = clean(data[8])
	u.Email = emailNameSanitizer(clean(u.NickName+"."+u.LastName)) + "@some.domain.com"
	if data[9] == "Kyllä" {
		u.Archieved = true
	} else {
		u.Archieved = false
	}
	return u
}

func getNewWilmaAccounts() ([]config.WilmaUser, error) {
	query := pq.PrimusQuery{}
	if len(os.Args) == 2 {
		if os.Args[1] == "all" {
			query = config.NewWilmaAccountsAll()
		} else if os.Args[1] == "yh" {
			query = config.NewWilmaAccountsYH()
		} else {
			return []config.WilmaUser{}, errors.New("illegal option")
		}
	}

	c := config.GetPrimusConfig()
	query.Host = c.PrimusHost
	query.Port = c.PrimusPort
	query.User = c.PrimusUser
	query.Pass = c.PrimusPassword
	pq.Debug = debugState
	output, err := pq.ExecuteAndRead(query, 60)
	if err != nil {
		return []config.WilmaUser{}, err
	}
	if output == "" {
		return []config.WilmaUser{}, nil
	}
	rows, err := readCSVFromString(output)
	if err != nil {
		return []config.WilmaUser{}, err
	}
	newAccounts := []config.WilmaUser{}
	for _, row := range rows {
		if len(row) == 10 {
			newUser := mapWilmaUsers(row)
			newAccounts = append(newAccounts, newUser)
			newUser = config.WilmaUser{}
		} else {
			log.Printf("Malformed and skipped data row: %s", row)
		}
	}
	return newAccounts, nil
}

var (
	// Version for build
	Version string
	// Build for build
	Build      string
	jobName    = "service-wilma-accounts"
	Env        string
	debugState bool
	table      string
)

func main() {
	c := config.GetPrimusConfig()
	debugState = false
	debug.SetGCPercent(100)

	start := time.Now()

	log.Printf("Target table is %s and port %s", table, c.PrimusPort)
	newWilmaAccounts, err := getNewWilmaAccounts()
	newWilmaAccountsCount := len(newWilmaAccounts)
	if err != nil {
		newWilmaAccountsCount = 0
		log.Print("Getting new Wilma account count failed.")
	} else {
		log.Printf("New Wilma account count: %d", newWilmaAccountsCount)
	}
	sqlDb := mssqlutils.ConnectOrDie(os.Getenv("SQL_SERVER"), os.Getenv("SQL_PORT"), config.SQLUser, config.SQLPassword, config.SQLDb, true, true)

	for _, account := range newWilmaAccounts {
		Email := ""
		err = nil
		if account.PersonalID != "" {
			Email, err = personLookup(sqlDb, account)
		}
		if err != nil {
			log.Printf("Person %s, Email: %s Err: %s", account.PersonalID, account.PersonalEmail, err.Error())
		} else {
			if Email != "" { // existing account
				log.Printf("Existing riveria account: %s", Email)
				account.Email = Email
				// todo try to find old account and update student id
				createNewWilmaAccountPrimus(account)
				if err != nil {
					log.Printf("Wilma user: %v, error: %s", account, err.Error())
				}
				err := linkNewWilmaAccountToStudent(account)
				if err != nil {
					log.Printf("Wilma user: %v, error: %s", account, err.Error())
				}
			} else { // create new account
				chosenEmail, err := newWilmaAccount(sqlDb, account)
				log.Printf("The new riveria account: %s", chosenEmail)
				if err != nil {
					log.Printf("Wilma user: %v, error: %s", account, err.Error())
				} else {
					account.Email = chosenEmail
					err = createNewWilmaAccountPrimus(account)
					if err != nil {
						log.Printf("Wilma user: %v, error: %s", account, err.Error())
					}
					err := linkNewWilmaAccountToStudent(account)
					if err != nil {
						log.Printf("Wilma user: %v, error: %s", account, err.Error())
					}
				}
			}
		}
	}

	t := time.Now()
	elapsed := t.Sub(start)
	log.Printf("Elapsed processing time %d.", elapsed)
	if err != nil {
		log.Printf("Disconnecting database session failed: %s", err)
	} else {
		log.Print("Disconnecting database session succeed.")
	}
}
