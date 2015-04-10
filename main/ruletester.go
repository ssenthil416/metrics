// program which takes
// - a rule file
// - a sample list of metrics
// and sees how well the rule performs against the metrics.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"square/vis/metrics-indexer/api"
	"square/vis/metrics-indexer/internal"
)

var (
	yamlFile    = flag.String("yaml-file", "", "Location of YAML configuration file.")
	metricsFile = flag.String("metrics-file", "", "Location of YAML configuration file.")
)

func exitWithRequired(flagName string) {
	fmt.Fprintf(os.Stderr, "%s is required\n", flagName)
	os.Exit(1)
}

func exitWithMessage(message string) {
	fmt.Fprint(os.Stderr, message)
	os.Exit(1)
}

func readRule(filename string) *internal.RuleSet {
	file, err := os.Open(filename)
	if err != nil {
		exitWithMessage("No rule file")
	}
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		exitWithMessage("Cannot read the rule YAML")
	}
	rule, err := internal.LoadYAML(bytes)
	if err != nil {
		exitWithMessage("Cannot parse Rule file")
	}
	return &rule
}

// Statistics represents the aggregated result of rules
// after running through the test file.
type Statistics struct {
	perMetric map[api.MetricKey]PerMetricStatistics
	matched   int // number of matched rows
	unmatched int // number of unmatched rows
}

// PerMetricStatistics represents per-metric result of rules
// after running through the test file.
type PerMetricStatistics struct {
	matched          int // number of matched rows
	reverseSuccess   int // number of reversed entries
	reverseError     int // number of incorrectly reversed entries.
	reverseIncorrect int // number of incorrectly reversed entries.
}

func main() {
	flag.Parse()
	if *yamlFile == "" {
		exitWithRequired("yaml-file")
	}
	if *metricsFile == "" {
		exitWithRequired("metrics-file")
	}
	ruleset := readRule(*yamlFile)
	metricFile, err := os.Open(*metricsFile)
	if err != nil {
		exitWithMessage("No metric file.")
	}
	scanner := bufio.NewScanner(metricFile)
	stat := Statistics{
		perMetric: make(map[api.MetricKey]PerMetricStatistics),
	}
	for scanner.Scan() {
		input := scanner.Text()
		converted, matched := ruleset.MatchRule(input)
		if matched {
			stat.matched++
			perMetric := stat.perMetric[converted.MetricKey]
			perMetric.matched++
			reversed, err := ruleset.Reverse(converted)
			if err != nil {
				perMetric.reverseError++
			} else if string(reversed) != input {
				perMetric.reverseIncorrect++
			} else {
				perMetric.reverseSuccess++
			}
			stat.perMetric[converted.MetricKey] = perMetric
		} else {
			stat.unmatched++
		}
	}
	total := stat.matched + stat.unmatched
	fmt.Printf("Processed %d entries\n", total)
	fmt.Printf("Matched:   %d\n", stat.matched)
	fmt.Printf("Unmatched: %d\n", stat.unmatched)
	fmt.Printf("Per-rule statistics\n")
	keys := ruleset.AllKeys()
	rowformat := "%-50s %7d %7d %7d %7d\n"
	headformat := "%-50s %7s %7s %7s %7s\n"
	fmt.Printf(headformat, "name", "match", "rev-suc", "rev-err", "rev-fail")
	for _, key := range keys {
		perMetric := stat.perMetric[key]
		fmt.Printf(rowformat,
			string(key),
			perMetric.matched,
			perMetric.reverseSuccess,
			perMetric.reverseError,
			perMetric.reverseIncorrect,
		)
	}
}
