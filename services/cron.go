package services

import (
	"log"

	cron "github.com/robfig/cron/v3"
)

func RunCron() {
	log.Print("Initiated cron")
	c := cron.New()
	c.AddFunc("* * * * *", func() { // Runs every minute
		log.Println("Cron running")
		// jobs, err := GetDueJobs() // Implement this function
		// if err != nil {
		// 	log.Println("Error fetching jobs:", err)
		// 	return
		// }
		// Fetch(jobs)
	})
	c.Start()
}
