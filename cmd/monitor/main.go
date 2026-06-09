package main

import (
	"fmt"

	"github.com/sakashimaa/site-monitor/internal/checker"
)

func main() {
	siteUrls := []string{
		"https://unavailable-123-qwerty-site",
		"https://microsoft.com",
		"https://jajahaha123123.qweqwe",
		"https://google.com",
		"https://github.com",
		"https://gmail.com",
		"https://dododo11111111yyttt.com",
		"https://amd.com",
		"https://nvidia.com",
		"https://vk.com",
	}

	for _, url := range siteUrls {
		res := checker.CheckSite(url)
		if !res.AvailableStatus {
			fmt.Printf("Site %s NOT ok\n", url)
			continue
		}

		fmt.Printf("Site %s ok\n", url)
	}
}
