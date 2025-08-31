package scrappergo

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const saraminJobSearchUrl = "https://www.saramin.co.kr/zf_user/search/recruit?=&recruitSort=relation"
const saraminJobDetailUrl = "https://www.saramin.co.kr/zf_user/jobs/relay/view?isMypage=no&rec_idx="

type extractedJob struct {
	id       string
	title    string
	location string
	salary   string
	summary  string
}

// Scrape Saramin by a term
func Scrape(term string) {
	var jobs []extractedJob
	totalPages := getPages()
	mainC := make(chan []extractedJob)
	log.Println(totalPages)

	searchWord := term
	pageNum := 1
	pageCount := 40

	for i := 1; i <= totalPages; i++ {
		go getPage(searchWord, pageNum, pageCount, mainC)
	}

	for i := 1; i <= totalPages; i++ {
		extractedJobs := <-mainC
		jobs = append(jobs, extractedJobs...)
	}

	wrtieJobs(jobs)
	fmt.Println("Done, extracted", len(jobs))
}

func wrtieJobs(jobs []extractedJob) {
	file, err := os.Create("jobs.csv")
	checkErr(err)
	w := csv.NewWriter(file)

	defer w.Flush()

	headers := []string{"ID", "Title", "Location", "Salary", "Summary"}
	wErr := w.Write(headers)
	checkErr(wErr)

	for _, job := range jobs {
		jobSlice := []string{saraminJobDetailUrl + job.id, job.title, job.location, job.salary, job.summary}
		jwErr := w.Write(jobSlice)
		checkErr(jwErr)
	}
}

func getPage(searchWord string, pageNum int, pageCount int, mainC chan<- []extractedJob) {
	var jobs []extractedJob
	c := make(chan extractedJob)
	url := saraminJobSearchUrl + "&searchword=" + searchWord + "&recruitPage=" + strconv.Itoa(pageNum) + "&recruitPageCount=" + strconv.Itoa(pageCount)

	fmt.Println(url)

	res, err := http.Get(url)
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)
	searchCards := doc.Find(".item_recruit")
	searchCards.Each(func(i int, card *goquery.Selection) {
		go extractJob(card, c)
	})

	for i := 0; i < searchCards.Length(); i++ {
		job := <-c
		jobs = append(jobs, job)
	}

	mainC <- jobs
}

func extractJob(card *goquery.Selection, c chan<- extractedJob) {
	id, _ := card.Attr("value")

	title := card.Find(".job_tit > a").Text()

	location := card.Find(".job_condition > span:first-child").Text()

	salaryText := card.Find(".job_condition > span:last-child").Text()
	salary := ""
	if strings.HasSuffix(salaryText, "만원") {
		salary = salaryText
	}

	summary := card.Find(".job_sector").Text()
	c <- extractedJob{
		id:       CleanString(id),
		title:    CleanString(title),
		location: CleanString(location),
		salary:   CleanString(salary),
		summary:  CleanString(summary),
	}
}

func CleanString(str string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(str)), " ")
}

func getPages() int {
	pages := 0
	res, err := http.Get("https://www.saramin.co.kr/zf_user/search/recruit?=&searchword=android&recruitPage=1&recruitSort=relation&recruitPageCount=40")
	checkErr(err)
	checkCode(res)

	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	checkErr(err)

	doc.Find(".pagination").Each(func(i int, s *goquery.Selection) {
		pages = s.Find("a").Length()
	})

	return pages
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func checkCode(res *http.Response) {
	if res.StatusCode != 200 {
		log.Fatalln("Request failed with status:", res.StatusCode)
	}
}
