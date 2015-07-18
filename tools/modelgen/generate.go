/***
Copyright 2014 Cisco Systems Inc. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"errors"
	"strings"
	"regexp"
	"unicode"
	"unicode/utf8"

	// log "github.com/Sirupsen/logrus"
)


// GenerateGoStructs generates go code from a schema
func (s *Schema) GenerateGoStructs() (string, error) {
	var goStr string
	for _, obj := range s.Objects {
		objStr, err := obj.GenerateGoStructs()
		if err == nil {
			goStr = goStr + objStr
		}
	}

	return goStr, nil
}

func (obj *Object) GenerateGoStructs() (string, error) {
	var goStr string

	objName := initialCap(obj.Name)
	goStr = goStr + fmt.Sprintf("type %s struct {\n", objName)

	// every object has a key
	goStr = goStr + fmt.Sprintf("	Key		string\n")

	// Walk each property and generate code for it
	for _, prop := range obj.Properties {
		propStr, err := prop.GenerateGoStructs()
		if err == nil {
			goStr = goStr + propStr
		}
	}

	// add link-sets
	if (len(obj.LinkSets) > 0) {
		goStr = goStr + fmt.Sprintf("	LinkSets	%sLinkSets\n", objName)
	}

	// add links
	if (len(obj.Links) > 0) {
		goStr = goStr + fmt.Sprintf("	Links	%sLinks\n", objName)
	}

	goStr = goStr + fmt.Sprintf("}\n")

	// define object's linkset
	if (len(obj.LinkSets) > 0) {
		goStr = goStr + fmt.Sprintf("type %sLinkSets struct {\n", objName)
		for lsName, ls := range obj.LinkSets {
			goStr = goStr + fmt.Sprintf("	%s	[]%sLinkSet\n", initialCap(lsName), ls.Name)
		}
		goStr = goStr + fmt.Sprintf("}\n")
	}

	// Define each link-sets
	for _, linkSet := range obj.LinkSets {
		subStr, err := linkSet.GenerateGoStructs()
		if err == nil {
			goStr = goStr + subStr
		}
	}

	// Define object's links
	if (len(obj.Links) > 0) {
		goStr = goStr + fmt.Sprintf("type %sLinks struct {\n", objName)
		for lName, link := range obj.Links {
			goStr = goStr + fmt.Sprintf("	%s	%sLink\n", initialCap(lName), link.Name)
		}
		goStr = goStr + fmt.Sprintf("}\n")
	}

	// define each link
	for _, link := range obj.Links {
		subStr, err := link.GenerateGoStructs()
		if err == nil {
			goStr = goStr + subStr
		}
	}

	return goStr, nil
}

func (ls *LinkSet) GenerateGoStructs() (string, error) {
	var goStr string

	goStr = goStr + fmt.Sprintf("type %sLinkSet struct {\n", ls.Name)
	goStr = goStr + fmt.Sprintf("	Type	string\n")
	goStr = goStr + fmt.Sprintf("	Key		string\n")
	goStr = goStr + fmt.Sprintf("	%s		*%s\n", ls.Ref, initialCap(ls.Ref))
	goStr = goStr + fmt.Sprintf("}\n")

	return goStr, nil
}

func (link *Link) GenerateGoStructs() (string, error) {
	var goStr string

	goStr = goStr + fmt.Sprintf("type %sLink struct {\n", link.Name)
	goStr = goStr + fmt.Sprintf("	Type	string\n")
	goStr = goStr + fmt.Sprintf("	Key		string\n")
	goStr = goStr + fmt.Sprintf("	%s		*%s\n", link.Ref, initialCap(link.Ref))
	goStr = goStr + fmt.Sprintf("}\n")

	return goStr, nil
}

func (prop *Property) GenerateGoStructs() (string, error) {
	var goStr string

	goStr = fmt.Sprintf("	%s	", prop.Name)
	switch prop.Type {
		case "string":
			goStr = goStr + fmt.Sprintf("string\n",)
		case "number":
			goStr = goStr + fmt.Sprintf("float64\n",)
		case "integer":
			goStr = goStr + fmt.Sprintf("int64\n",)
		case "bool":
			goStr = goStr + fmt.Sprintf("bool\n",)
		default:
			return "", errors.New("Unknown Property")
	}

	return goStr, nil
}

var (
	newlines  = regexp.MustCompile(`(?m:\s*$)`)
	acronyms  = regexp.MustCompile(`(Url|Http|Id|Io|Uuid|Api|Uri|Ssl|Cname|Oauth|Otp)$`)
	camelcase = regexp.MustCompile(`(?m)[-.$/:_{}\s]`)
)

func initialCap(ident string) string {
	if ident == "" {
		panic("blank identifier")
	}
	return depunct(ident, true)
}

func initialLow(ident string) string {
	if ident == "" {
		panic("blank identifier")
	}
	return depunct(ident, false)
}

func depunct(ident string, initialCap bool) string {
	matches := camelcase.Split(ident, -1)
	for i, m := range matches {
		if initialCap || i > 0 {
			m = capFirst(m)
		}
		matches[i] = acronyms.ReplaceAllStringFunc(m, func(c string) string {
			if len(c) > 4 {
				return strings.ToUpper(c[:2]) + c[2:]
			}
			return strings.ToUpper(c)
		})
	}
	return strings.Join(matches, "")
}

func capFirst(ident string) string {
	r, n := utf8.DecodeRuneInString(ident)
	return string(unicode.ToUpper(r)) + ident[n:]
}
