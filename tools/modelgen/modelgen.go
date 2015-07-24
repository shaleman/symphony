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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

var (
	source = flag.String("s", "./", "Location of json schema")
	output = flag.String("o", "", "Output directory")
)

func main() {
	flag.Parse()

	var schema *Schema

	// Parse all files in input directory
	err := filepath.Walk(*source, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore non-json files
		if filepath.Ext(path) != ".json" {
			return nil
		}

		fmt.Printf("Parsing file: %s\n", path)

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// Parse the schema
		sch, err := ParseSchema(b)
		if err != nil {
			return err
		}

		// Append to global schema
		schema = MergeSchema(schema, sch)

		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Generate file headers
	outStr := schema.GenerateGoHdrs()

	// Generate structs
	structStr, err := schema.GenerateGoStructs()
	if err != nil {
		log.Fatalf("Error generating go structs. Err: %v", err)
	}

	// Merge the header and struct
	outStr = outStr + structStr

	// Merge rest handler
	outStr = outStr + schema.GenerateGoFuncs()

	outPath := "./"
	if *output != "" {
		outPath = *output
	}

	// Write the Go file output
	goFileName := path.Join(outPath, schema.Name + ".go")
	file, err := os.Create(goFileName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Writing to file: %s\n", goFileName)

	fmt.Fprintln(file, outStr)

	// Generate javascript
	outStr, err = schema.GenerateJs()
	if err != nil {
		log.Fatalf("Error generating javascript. Err: %v", err)
	}

	// Write javascript file
	jsFileName := path.Join(outPath, schema.Name + ".js")
	file, err = os.Create(jsFileName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Writing to file: %s\n", jsFileName)

	fmt.Fprintln(file, outStr)
}
