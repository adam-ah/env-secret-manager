package main

import "strings"

const envExampleTitle = "INJECT AS AN ENV VARIABLE"
const envCurlExampleTitle = "CURL USING THE ENV VARIABLE"
const curlExampleTitle = "INJECT INTO CURL"

type secretExamples struct {
	reference string
	name      string
}

var sampleExamples = secretExamples{reference: "my-app:SAMPLE_SECRET", name: "SAMPLE_SECRET"}

func (e secretExamples) envDescription() string {
	return "Available as $" + e.name + " in the child process"
}

func (e secretExamples) variableCommand() string {
	if e.name == "" {
		return ""
	}
	return "secretenv exec " + shellArgument(e.reference) + " -- your-program"
}

func (e secretExamples) curlCommand() string {
	if e.name == "" {
		return ""
	}
	return strings.Join([]string{
		"secretenv exec " + shellArgument(e.reference) + " -- curl \\",
		"  --header " + shellQuote("Authorization: Bearer {"+e.name+"}") + " \\",
		"  https://api.example.com",
	}, "\n")
}

func (e secretExamples) envCurlCommand() string {
	if e.name == "" {
		return ""
	}
	command := strings.Join([]string{
		"exec curl --silent --show-error \\",
		"  --header \"Authorization: Bearer $" + e.name + "\" \\",
		"  https://api.example.com",
	}, "\n")
	return "secretenv exec " + shellArgument(e.reference) + " -- sh -c " + shellQuote(command)
}
