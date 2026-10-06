package models

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/check.v1"
)

func (s *ModelsSuite) TestAttachment(c *check.C) {
	ptx := PhishingTemplateContext{
		BaseRecipient: BaseRecipient{
			FirstName: "Foo",
			LastName:  "Bar",
			Email:     "foo@bar.com",
			Position:  "Space Janitor",
		},
		BaseURL:     "http://testurl.com",
		URL:         "http://testurl.com/?rid=1234567",
		TrackingURL: "http://testurl.local/track?rid=1234567",
		Tracker:     "<img alt='' style='display: none' src='http://testurl.local/track?rid=1234567'/>",
		From:        "From Address",
		RId:         "1234567",
	}

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		log.Fatalf("Failed to open attachment folder 'testdata': %v\n", err)
	}
	for _, ff := range files {
		if !ff.IsDir() && !strings.Contains(ff.Name(), "templated") {
			fname := ff.Name()
			fmt.Printf("Checking attachment file -> %s\n", fname)
			data := readFile("testdata/" + fname)
			if filepath.Ext(fname) == ".b64" {
				fname = fname[:len(fname)-4]
			}
			a := Attachment{
				Content: data,
				Name:    fname,
			}
			t, err := a.ApplyTemplate(ptx)
			c.Assert(err, check.Equals, nil)
			c.Assert(a.vanillaFile, check.Equals, strings.Contains(fname, "without-vars"))
			c.Assert(a.vanillaFile, check.Not(check.Equals), strings.Contains(fname, "with-vars"))

			// Verify template was applied as expected
			tt, err := ioutil.ReadAll(t)
			if err != nil {
				log.Fatalf("Failed to parse templated file '%s': %v\n", fname, err)
			}
			templatedFile := base64.StdEncoding.EncodeToString(tt)
			expectedOutput := readFile("testdata/" + strings.TrimSuffix(ff.Name(), filepath.Ext(ff.Name())) + ".templated" + filepath.Ext(ff.Name())) // e.g text-file-with-vars.templated.txt
			switch filepath.Ext(fname) {
			case ".docx", ".docm", ".pptx", ".xlsx", ".xlsm":
				// Compression output can change between Go versions. Verify every
				// archive entry and its exact contents instead of compressed bytes.
				expectedBytes, err := base64.StdEncoding.DecodeString(expectedOutput)
				c.Assert(err, check.IsNil)
				actualZip, err := zip.NewReader(bytes.NewReader(tt), int64(len(tt)))
				c.Assert(err, check.IsNil)
				expectedZip, err := zip.NewReader(bytes.NewReader(expectedBytes), int64(len(expectedBytes)))
				c.Assert(err, check.IsNil)
				c.Assert(len(actualZip.File), check.Equals, len(expectedZip.File))
				for i, expectedFile := range expectedZip.File {
					actualFile := actualZip.File[i]
					c.Assert(actualFile.Name, check.Equals, expectedFile.Name)
					c.Assert(attachmentEntryContents(c, actualFile), check.DeepEquals,
						attachmentEntryContents(c, expectedFile), check.Commentf("%s: %s", fname, expectedFile.Name))
				}
			default:
				c.Assert(templatedFile, check.Equals, expectedOutput)
			}
		}
	}
}

func readFile(fname string) string {
	f, err := os.Open(fname)
	if err != nil {
		log.Fatalf("Failed to open file '%s': %v\n", fname, err)
	}
	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatalf("Failed to read file '%s': %v\n", fname, err)
	}
	data := ""
	if filepath.Ext(fname) == ".b64" {
		data = string(content)
	} else {
		data = base64.StdEncoding.EncodeToString(content)
	}
	return data
}

func attachmentEntryContents(c *check.C, file *zip.File) []byte {
	reader, err := file.Open()
	c.Assert(err, check.IsNil)
	defer reader.Close()
	contents, err := ioutil.ReadAll(reader)
	c.Assert(err, check.IsNil)
	return contents
}
