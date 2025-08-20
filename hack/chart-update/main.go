package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v50/github"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/helm/pkg/provenance"
)

const (
	indexFilePath = "../../index.yaml"
	gitHubOrgName = "kubernetes-sigs"
	repoName      = "cluster-api-operator"
)

// chartInfo represents information about a chart to be processed
type chartInfo struct {
	name        string
	description string
}

// List of charts to process
var charts = []chartInfo{
	{
		name:        "cluster-api-operator",
		description: "Cluster API Operator",
	},
	{
		name:        "cluster-api-operator-providers",
		description: "Cluster API Provider Custom Resources",
	},
}

func main() {
	fmt.Println("🚀 Starting index.yaml update tool")

	var tag string
	flag.StringVar(&tag, "release-tag", "", "github release tag")
	flag.Parse()

	if tag == "" {
		fmt.Println("❌ Please provide a release tag")
		os.Exit(1)
	}

	fmt.Println("⚙️  Loading index.yaml file from repo root directory")

	indexFile := loadIndexFile()

	fmt.Println("🔎 Finding chart archives in release assets")

	// Get all release assets once
	releaseAssets := getReleaseAssets(tag)

	// Process each chart
	processedCharts := 0
	for _, chartInfo := range charts {
		fmt.Printf("\n📊 Processing chart: %s\n", chartInfo.name)

		// Check if chart already exists in index
		if _, err := indexFile.Get(chartInfo.name, tag[1:]); err == nil {
			fmt.Printf("✅ Chart %s already exists in index file, skipping\n", chartInfo.name)
			continue
		}

		// Find chart asset
		chartAsset := findChartAsset(releaseAssets, chartInfo.name, tag)
		if chartAsset == nil {
			fmt.Printf("⚠️  Chart archive for %s not found in release assets, skipping\n", chartInfo.name)
			continue
		}

		fmt.Printf("📦 Downloading %s chart archive to a temp directory\n", chartInfo.name)
		archivePath, chart := downloadChart(chartAsset)

		fmt.Printf("👉🏻 Adding %s entry to index.yaml\n", chartInfo.name)
		addEntryToIndexFile(indexFile, chartAsset, archivePath, chart)

		processedCharts++
	}

	if processedCharts == 0 {
		fmt.Println("\n⚠️  No new charts were added to index.yaml")
		os.Exit(0)
	}

	fmt.Println("\n📝 Writing index.yaml file to repo root directory")

	if err := indexFile.WriteFile(indexFilePath, 0644); err != nil {
		fmt.Println("❌ Error writing index file: ", err)
		os.Exit(1)
	}

	fmt.Printf("\n✅ Done updating index.yaml file. Added %d chart(s)\n", processedCharts)
}

func loadIndexFile() *repo.IndexFile {
	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		fmt.Println("❌ Error loading index file: ", err)
		os.Exit(1)
	}

	return indexFile
}

func getReleaseAssets(tag string) []*github.ReleaseAsset {
	ghClient := github.NewClient(nil)

	release, _, err := ghClient.Repositories.GetReleaseByTag(context.TODO(), gitHubOrgName, repoName, tag)
	if err != nil {
		fmt.Println("❌ Error getting github release: ", err)
		os.Exit(1)
	}

	return release.Assets
}

func findChartAsset(assets []*github.ReleaseAsset, chartName, tag string) *github.ReleaseAsset {
	expectedFileName := fmt.Sprintf("%s-%s.tgz", chartName, tag[1:])

	for _, asset := range assets {
		if *asset.Name == expectedFileName {
			return asset
		}
	}

	return nil
}

func downloadChart(chartAsset *github.ReleaseAsset) (string, *chart.Chart) {
	tempDirPath, err := os.MkdirTemp("", "")
	if err != nil {
		fmt.Println("❌ Error creating temp dir: ", err)
		os.Exit(1)
	}

	archivePath := filepath.Join(tempDirPath, *chartAsset.Name)

	resp, err := http.Get(*chartAsset.BrowserDownloadURL)
	if err != nil {
		fmt.Println("❌ Error downloading chart archive: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	out, err := os.Create(archivePath)
	if err != nil {
		fmt.Println("❌ Error creating chart archive: ", err)
		os.Exit(1)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		fmt.Println("❌ Error copying chart archive: ", err)
		os.Exit(1)
	}

	chart, err := loader.LoadFile(archivePath)
	if err != nil {
		fmt.Println("❌ Error loading chart: ", err)
		os.Exit(1)
	}

	return archivePath, chart
}

func addEntryToIndexFile(indexFile *repo.IndexFile, chartAsset *github.ReleaseAsset, archivePath string, chart *chart.Chart) {
	s := strings.Split(*chartAsset.BrowserDownloadURL, "/") // https://github.com/helm/chart-releaser/blob/main/pkg/releaser/releaser.go#L299
	s = s[:len(s)-1]

	hash, err := provenance.DigestFile(archivePath)
	if err != nil {
		fmt.Println("❌ Error generating hash: ", err)
		os.Exit(1)
	}

	if err := indexFile.MustAdd(chart.Metadata, filepath.Base(archivePath), strings.Join(s, "/"), hash); err != nil {
		fmt.Println("❌ Error adding to index file: ", err)
		os.Exit(1)
	}

	indexFile.SortEntries()
	indexFile.Generated = time.Now()
}
