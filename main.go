package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/joho/godotenv"
)

type Person struct {
	Name  string
	Email string
}

const sleepTimeBase int = 5
const sleepTimeVariation int = 10
const testEmailsDir string = "test-emails"
const notSetEmailSubjectDefault string = "NOTSET"

func main() {
	// since this is published on Github and can be used to spam people,
	// print some stuff to make it clear using the program is their choice
	// and responsibility is all theirs
	checkFirstRun()

	// start time to print overall program run duration
	startTime := time.Now()

	// load the env variables
	err := godotenv.Load("emailer.env")
	if err != nil {
		log.Fatal("no .env file was found, so cannot run")
	}

	verifyEnvVariablesSet()

	// get command line flags to determine whether to do a test run or real run
	// and to set the subject line
	runFlag := flag.Bool("run", false, "add -run to make the emailer run")
	subjectFlag := flag.String("subject", notSetEmailSubjectDefault, "the subject line - REQUIRED")
	// todo
	// helpFlag := flag.Bool("help", false, "show a help screen")
	flag.Parse()

	// kill the program if the subject line isn't set
	if *subjectFlag == notSetEmailSubjectDefault {
		log.Fatal("you MUST set the subject line here to run by using -subject=\"subject\"")
	}

	// get this emailer's dialer from info in .env file
	dialer := getSmtpDialer()

	emailList := getPeopleListFromCsv()

	// delete the test-emails folder and all files if running a test run (so new ones can be added)
	if !*runFlag {
		os.RemoveAll(testEmailsDir)
	}

	// iterate through people
	for i, person := range emailList {
		if *runFlag {
			sendEmail(person, dialer, *subjectFlag)
			// sleep between emails
			// but skip the waiting after the last one
			if i != (len(emailList) - 1) {
				waitTime := getEmailDelayTime()
				fmt.Println("waiting for ", waitTime)
				time.Sleep(waitTime)
			}
		} else {
			// print info for test run
			fmt.Println("saving email # " + fmt.Sprint(i) + " to " + person.Email + " from: " + os.Getenv("EMAIL_FROM_EMAIL") + " subject: " + *subjectFlag)
			saveExampleEmail(person, i)
		}
	}

	if !*runFlag {
		// print a line saying the test was run and should run with -run set to actually run the emailer
		fmt.Println("this was a test run, so all files are in the test folder. Run again with -run set to actually run once.")
	} else {
		// this was a real run, so print the amount of time this took
		duration := time.Since(startTime)

		fmt.Println("\n\n🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉")
		fmt.Println("Finished sending ", len(emailList), " emails. It took ", math.Round(duration.Minutes()), " minute(s)")
	}
}

func getPeopleListFromCsv() []Person {
	finalList := []Person{}

	// verify emails are probably valid with this regex
	emailRegex := regexp.MustCompile(`^[\w-\.]+@([\w-]+\.)+[\w-]{2,4}$`)

	// open the csv file
	f, err := os.Open("list.csv")
	if err != nil {
		log.Fatal("error opening the list.csv file\n", err)
	}

	// read the csv file
	csvReader := csv.NewReader(f)

	// iterate through and add the person stuff to the list
	// for statement here doesn't need a condition since the
	// loop will break when the reader hits a end of file
	for i := 0; ; i++ {
		record, err := csvReader.Read()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal("error reading a line from the csv file:\n", err)
		}

		name := strings.TrimSpace(record[0])
		email := strings.TrimSpace(record[1])

		// check data
		if (len(name) < 1) || (len(email) < 1) {
			log.Fatal("there's an error with one of the fields. Either name or email are missing. Row: ", i+1)
		}

		if emailRegex.MatchString(email) {
			// email appears to be valid, so add to the list
			finalList = append(finalList, Person{name, email})
		} else {
			log.Fatal("there appears to be an invalid email at row ", i+1, " so check that. Email is listed as: ", email)
		}
	}

	return finalList
}

func saveExampleEmail(person Person, i int) {
	emailBody := getEmailBodyString(person)

	// create the folder
	_, err := os.Stat("test-emails")
	if err != nil {
		// folder likely doesn't exist so create it
		err = os.Mkdir(testEmailsDir, 0700)
		if err != nil {
			log.Fatal("error creating the test emails directory:\n", err)
		}
	}

	// create the file
	f, err := os.Create(testEmailsDir + "/" + fmt.Sprint(i) + ".html")
	if err != nil {
		log.Fatal("error creating a file for test emails:\n", err)
	}
	defer f.Close()

	_, err = f.WriteString(emailBody)
	if err != nil {
		log.Fatal("error writing email to file:\n", err)
	}
}

func sendEmail(person Person, dialer *gomail.Dialer, subject string) {
	emailBody := getEmailBodyString(person)

	// construct and send email
	m := gomail.NewMessage()
	m.SetHeader("From", formattedEmailFromString())
	m.SetAddressHeader("To", person.Email, person.Name)
	m.SetHeader("Subject", subject)

	// if need to set cc, do so
	cc := getAllCCRecipients()
	if len(cc) == 2 {
		m.SetAddressHeader("Cc", cc[0], cc[1])
	}

	// set the body
	m.SetBody("text/html", emailBody)

	// send
	if err := dialer.DialAndSend(m); err != nil {
		panic(err)
	}

	fmt.Println("successfully sent an email to " + person.Email)
}

// pointless to have a function here, but I think it makes things easier to read
func formattedEmailFromString() string {
	return os.Getenv("EMAIL_FROM_NAME") + " <" + os.Getenv("EMAIL_FROM_EMAIL") + ">"
}

func getSmtpDialer() *gomail.Dialer {
	// get the email login stuff
	emailFrom := os.Getenv("EMAIL_FROM_EMAIL")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")

	// convert the port to an int
	i, err := strconv.Atoi(smtpPort)
	if err != nil {
		log.Fatal("error converting the saved port in the .env file to a number")
	}

	return gomail.NewDialer(smtpHost, i, emailFrom, emailPassword)
}

// just return the cc recipient info
// saved in the .env file
// in format:
// ccRecipient[0] = email
// ccRecipient[1] = name
func getAllCCRecipients() []string {
	ccEmail := os.Getenv("CC_PERSON")
	ccName := os.Getenv("CC_NAME")

	if (len(ccEmail) > 1) && (len(ccName) > 1) {
		return []string{ccEmail, ccName}
	} else {
		return []string{}
	}
}

// getEmailBodyString takes a person struct then returns a string
// of the formatted email with the info provided
func getEmailBodyString(person Person) string {
	var formattedEmail bytes.Buffer

	tmpl, err := template.ParseFiles("./email.html")

	if err != nil {
		log.Fatal("there was an error making the template: ", err)
	}

	err = tmpl.Execute(&formattedEmail, person)
	if err != nil {
		log.Fatal("error executin the statement ", err)
	}

	return formattedEmail.String()
}

// obvious, but just checks all env variables are set before allowing the program to run
func verifyEnvVariablesSet() {
	envList := []string{"EMAIL_FROM_EMAIL", "EMAIL_FROM_NAME", "EMAIL_PASSWORD", "SMTP_HOST", "SMTP_PORT"}

	for _, variable := range envList {
		if len(os.Getenv(variable)) <= 1 {
			log.Fatal("one variable hasn't been set in the emailer.env file, check the file again (" + variable + " not set) - no emails sent")
		}
	}
}

// this returns a duration for the loop to wait x seconds before
// sending the next email
// just to avoid looking too much like a bot or spammer
// the program gets a random value between 0-10
// and adds to 5
// so the delay between emails is 5-15 seconds
func getEmailDelayTime() time.Duration {
	wait := sleepTimeBase + rand.Intn(sleepTimeVariation)
	return time.Second * time.Duration(wait)
}

// not sure if this is necessary or not, but just want to be careful...
func checkFirstRun() {
	_, err := os.Stat(".firstrun")
	if errors.Is(err, os.ErrNotExist) {
		// the file doesn't exist, so print a thing and create the file
		outputMessage := "*******************************************************************\n" +
			"It appears this is your first time using this program.\n\n" +
			"Please note that this software is provided \"AS IS\".\n\n" +
			"The author of this program will take no responsibility for others'\nmisuse, data loss, or any other result of using this program.\n\n" +
			"The author of this program also makes no guarantees the program\nwill work as expected.\n\n" +
			"By continuing to use this program, you accept that this program\nis provided \"AS IS\".\n\n" +
			"If you understand and accept full responsibility for using this\nprogram, type `yes` (without quotes) to continue.\n" +
			"*******************************************************************\n"

		fmt.Println("\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n" + outputMessage)

		var response string

		fmt.Scanln(&response)

		response = strings.TrimSpace(strings.ToLower(response))

		if response == "yes" {
			f, err := os.Create(".firstrun")
			if err != nil {
				log.Fatal("error creating first run file: ", err)
			}
			defer f.Close()

			fileContents := fmt.Sprint("user accepted at ", time.Now())

			f.WriteString(fileContents)
			fmt.Print("\n\nUser agreed, continuing....\n\n\n\n")
		} else {
			fmt.Println("User did not agree, exiting.")
			os.Exit(1)
		}
	}
}
