package config

import (
	"strconv"
	"time"

	"github.com/beevik/etree"
	pq "github.com/pasiol/gopq"
)

// Teachers query
func NewWilmaAccountsYH() pq.PrimusQuery {

	pq := pq.PrimusQuery{}
	pq.Charset = "UTF-8"
	pq.Database = "opphenk"
	pq.Sort = ""
	pq.Search = ""
	pq.Data = ""
	pq.Footer = ""

	return pq
}

func NewWilmaAccountsAll() pq.PrimusQuery {

	pq := pq.PrimusQuery{}
	pq.Charset = "UTF-8"
	pq.Database = "opphenk"
	pq.Sort = ""
	pq.Search = ""
	pq.Data = ""
	pq.Footer = ""

	return pq
}

// WilmaUser struct
type WilmaUser struct {
	ID            string
	UserType      string
	NickName      string
	FirstNames    string
	LastName      string
	PersonalEmail string
	PhoneNumber   string
	PersonalID    string
	StudentID     string
	Email         string
	Archieved     bool
}

func UserAccountXML(user WilmaUser) (string, error) {
	applicantDoc := etree.NewDocument()
	applicantDoc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	primusquery := applicantDoc.CreateElement("PRIMUSQUERY_IMPORT")
	primusquery.CreateAttr("ARCHIVEMODE", "0")
	primusquery.CreateAttr("CREATEIFNOTFOUND", "1")
	identity := primusquery.CreateElement("IDENTITY")
	identity.CreateText("service-create-wilma-accounts")
	card := primusquery.CreateElement("CARD")
	card.CreateAttr("FIND", user.Email)
	email := card.CreateElement("EMAIL")
	email.CreateText(user.Email)
	userAccount := card.CreateElement("TUNNUS")
	userAccount.CreateText(user.Email)
	lastName := card.CreateElement("SUKUNIMI")
	lastName.CreateText(user.LastName)
	firstName := card.CreateElement("ETUNIMET")
	firstName.CreateText(user.FirstNames)
	nickName := card.CreateElement("KUTSUMANIMI")
	nickName.CreateText(user.NickName)
	cellPhone := card.CreateElement("MATKAPUHELIN")
	cellPhone.CreateText(user.PhoneNumber)
	archieve := card.CreateElement("ARKISTO")
	archieve.CreateText("Ei")

	applicantDoc.Indent(2)
	xmlAsString, _ := applicantDoc.WriteToString()
	filename, err := pq.CreateTMPFile(pq.StringWithCharset(128)+".xml", xmlAsString)
	if err != nil {
		return "", err
	}
	return filename, nil
}

// UpdateAccountXML generator
func UpdateStudentXML(student WilmaUser) (string, error) {
	year := strconv.Itoa(time.Now().Year())[2:] // remember update this end of the year 9999
	updateDoc := etree.NewDocument()
	updateDoc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	primusquery := updateDoc.CreateElement("PRIMUSQUERY_IMPORT")
	primusquery.CreateAttr("ARCHIVEMODE", "0")
	primusquery.CreateAttr("CREATEIFNOTFOUND", "0")
	identity := updateDoc.CreateElement("IDENTITY")
	identity.CreateText("service-update-accounts")
	card := updateDoc.CreateElement("CARD")
	card.CreateAttr("FIND", student.ID)
	email2 := card.CreateElement("EMAIL2")
	email2.CreateText(student.Email)
	noLDAP := card.CreateElement("EILDAP")
	noLDAP.CreateText("Ei")
	userGroup := card.CreateElement("KÄYTTÄJÄRYHMÄ")
	userGroup.CreateAttr("CMD", "MODIFY")
	userGroup.CreateAttr("LINE", "1")
	userGroup.CreateText("Riverian opiskelijat")
	newUserAccount := card.CreateElement("UUSITUNNUS")
	newUserAccount.CreateAttr("CMD", "MODIFY")
	newUserAccount.CreateAttr("LINE", "1")
	newUserAccount.CreateText(student.Email)
	newTypeUserAccount := card.CreateElement("UUSITUNNUSKAYTOSSA")
	newTypeUserAccount.CreateText("Kyllä")
	studentNumber := card.CreateElement("OPISKELIJANUMERO")
	studentNumber.CreateText(string(year + student.ID))

	updateDoc.Indent(2)
	xmlAsString, _ := updateDoc.WriteToString()
	filename, err := pq.CreateTMPFile(pq.StringWithCharset(128)+".xml", xmlAsString)
	if err != nil {
		return "", nil
	}
	return filename, err
}
