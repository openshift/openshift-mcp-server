package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type validationItem struct {
	Errors      int    `json:"errors"`
	ObjectCount int    `json:"objectCount"`
	Warnings    int    `json:"warnings"`
	Namespace   string `json:"namespace"`
	Cluster     string `json:"cluster"`
}

func PrintValidationsList(raw string, jsonOut bool) {
	var items []validationItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		// If it is not JSON, print raw
		fmt.Println(raw)
		return
	}
	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(items)
		return
	}

	// Group by cluster
	byCluster := map[string][]validationItem{}
	clusters := make([]string, 0)
	for _, it := range items {
		byCluster[it.Cluster] = append(byCluster[it.Cluster], it)
	}
	for c := range byCluster {
		clusters = append(clusters, c)
	}
	sort.Strings(clusters)

	for _, c := range clusters {
		ns := byCluster[c]
		// Decide cluster color: red if any error, yellow if any warning, else green
		clusterColor := colorGreen
		for _, n := range ns {
			if n.Errors > 0 {
				clusterColor = colorRed
				break
			}
			if clusterColor != colorRed && n.Warnings > 0 {
				clusterColor = colorYellow
			}
		}
		fmt.Printf("%s%s%s\n", clusterColor, c, colorReset)
		// Sort namespaces by name
		sort.Slice(ns, func(i, j int) bool { return ns[i].Namespace < ns[j].Namespace })
		for _, n := range ns {
			// Decide per-namespace color severity
			nsColor := colorGreen
			if n.Errors > 0 {
				nsColor = colorRed
			} else if n.Warnings > 0 {
				nsColor = colorYellow
			}

			// Color only the namespace and the corresponding severity word
			coloredNS := fmt.Sprintf("%s%s%s", nsColor, n.Namespace, colorReset)
			errorsWord := "errors"
			warningsWord := "warnings"
			if n.Errors > 0 {
				errorsWord = colorRed + errorsWord + colorReset
			}
			if n.Warnings > 0 && n.Errors == 0 {
				warningsWord = colorYellow + warningsWord + colorReset
			}

			// Example: - default total(1) errors(0) warnings(1)
			fmt.Printf("  - %s total(%d) %s(%d) %s(%d)\n", coloredNS, n.ObjectCount, errorsWord, n.Errors, warningsWord, n.Warnings)
		}
		fmt.Println()
	}
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
)
