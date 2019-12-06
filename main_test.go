package main

import (
	"bytes"
	"testing"
)

func TestParseRequestBody(t *testing.T) {
	body := bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8"?>
<ns0:BatchList xmlns:ns0="http://naesb.org/espi"><ns0:resources>https://api.pge.com/GreenButtonConnect/espi/1_1/resource/Batch/Bulk/50952?correlationID=e043071d-d93d-4876-8907-c501e622a825</ns0:resources><ns0:resources>https://api.pge.com/GreenButtonConnect/espi/1_1/resource/Batch/Bulk/50952?correlationID=170c96ec-33b9-4cc1-9f6d-35f90472770d</ns0:resources></ns0:BatchList>`)
	resources := ParseRequestBody(body)
	if len(resources) != 2 {
		t.Error("Missing resources")
	}

	if resources[0].Value != "https://api.pge.com/GreenButtonConnect/espi/1_1/resource/Batch/Bulk/50952?correlationID=e043071d-d93d-4876-8907-c501e622a825" {
		t.Error("Incorrect URL for resource[0]")
	}

	if resources[0].Value != "https://api.pge.com/GreenButtonConnect/espi/1_1/resource/Batch/Bulk/50952?correlationID=170c96ec-33b9-4cc1-9f6d-35f90472770d" {
		t.Error("Incorrect URL for resource[1]")
	}
}
