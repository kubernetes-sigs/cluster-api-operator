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

	indexFile := loadIndexFile(tag)

	fmt.Println("🔎 Finding chart archive in release assets")

	chartAsset := findChartReleaseAsset(tag)

	fmt.Println("📦 Downloading chart archive to a temp directory")

	archivePath, chart := downloadChart(chartAsset)

	fmt.Println("👉🏻 Adding entry to index.yaml")
	addEntryToIndexFile(indexFile, chartAsset, archivePath, chart)

	fmt.Println("📝 Writing index.yaml file to repo root directory")

	if err := indexFile.WriteFile(indexFilePath, 0644); err != nil {
		fmt.Println("❌ Error writing index file: ", err)
		os.Exit(1)
	}

	fmt.Println("✅ Done updating index.yaml file")
}

func loadIndexFile(tag string) *repo.IndexFile {
	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		fmt.Println("❌ Error loading index file: ", err)
		os.Exit(1)
	}

	if _, err := indexFile.Get(repoName, tag[1:]); err == nil {
		fmt.Println("✅ Chart already exists in index file, no need to update")
		os.Exit(0)
	}

	return indexFile
}

func findChartReleaseAsset(tag string) *github.ReleaseAsset {
	ghClient := github.NewClient(nil)

	release, _, err := ghClient.Repositories.GetReleaseByTag(context.TODO(), gitHubOrgName, repoName, tag)
	if err != nil {
		fmt.Println("❌ Error getting github release: ", err)
		os.Exit(1)
	}

	chartAsset := &github.ReleaseAsset{}
	found := false
	for _, asset := range release.Assets {
		if *asset.Name == fmt.Sprintf("%s-%s.tgz", repoName, tag[1:]) {
			chartAsset = asset
			found = true
			break
		}
	}

	if !found {
		fmt.Printf("❌ Chart archive not found in release assets for release %s, please check if release was published correctly\n", tag)
		os.Exit(1)
	}

	return chartAsset
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
