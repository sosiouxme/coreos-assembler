// Copyright 2019 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.)

package config

import (
	"fmt"

	"github.com/coreos/butane/config/common"
	fcos1_0 "github.com/coreos/butane/config/fcos/v1_0"
	fcos1_1 "github.com/coreos/butane/config/fcos/v1_1"
	fcos1_2 "github.com/coreos/butane/config/fcos/v1_2"
	fcos1_3 "github.com/coreos/butane/config/fcos/v1_3"
	fcos1_4_exp "github.com/coreos/butane/config/fcos/v1_4_exp"
	openshift4_8 "github.com/coreos/butane/config/openshift/v4_8"
	openshift4_9_exp "github.com/coreos/butane/config/openshift/v4_9_exp"
	rhcos0_1 "github.com/coreos/butane/config/rhcos/v0_1"

	"github.com/coreos/go-semver/semver"
	"github.com/coreos/vcontext/report"
	"gopkg.in/yaml.v3"
)

var (
	registry = map[string]translator{}
)

/// Fields that must be included in the root struct of every spec version.
type commonFields struct {
	Version string `yaml:"version"`
	Variant string `yaml:"variant"`
}

func init() {
	RegisterTranslator("fcos", "1.0.0", fcos1_0.ToIgn3_0Bytes)
	RegisterTranslator("fcos", "1.1.0", fcos1_1.ToIgn3_1Bytes)
	RegisterTranslator("fcos", "1.2.0", fcos1_2.ToIgn3_2Bytes)
	RegisterTranslator("fcos", "1.3.0", fcos1_3.ToIgn3_2Bytes)
	RegisterTranslator("fcos", "1.4.0-experimental", fcos1_4_exp.ToIgn3_3Bytes)
	RegisterTranslator("openshift", "4.8.0", openshift4_8.ToConfigBytes)
	RegisterTranslator("openshift", "4.9.0-experimental", openshift4_9_exp.ToConfigBytes)
	RegisterTranslator("rhcos", "0.1.0", rhcos0_1.ToIgn3_2Bytes)
}

/// RegisterTranslator registers a translator for the specified variant and
/// version to be available for use by TranslateBytes.  This is only needed
/// by users implementing their own translators outside the Butane package.
func RegisterTranslator(variant, version string, trans translator) {
	key := fmt.Sprintf("%s+%s", variant, version)
	if _, ok := registry[key]; ok {
		panic("tried to reregister existing translator")
	}
	registry[key] = trans
}

func getTranslator(variant string, version semver.Version) (translator, error) {
	t, ok := registry[fmt.Sprintf("%s+%s", variant, version.String())]
	if !ok {
		return nil, fmt.Errorf("No translator exists for variant %s with version %s", variant, version.String())
	}
	return t, nil
}

// translators take a raw config and translate it to a raw Ignition config. The report returned should include any
// errors, warnings, etc and may or may not be fatal. If report is fatal, or other errors are encountered while translating
// translators should return an error.
type translator func([]byte, common.TranslateBytesOptions) ([]byte, report.Report, error)

// TranslateBytes wraps all of the individual TranslateBytes functions in a switch that determines the correct one to call.
// TranslateBytes returns an error if the report had fatal errors or if other errors occured during translation.
func TranslateBytes(input []byte, options common.TranslateBytesOptions) ([]byte, report.Report, error) {
	// first determine version. This will ignore most fields, so don't use strict
	ver := commonFields{}
	if err := yaml.Unmarshal(input, &ver); err != nil {
		return nil, report.Report{}, fmt.Errorf("Error unmarshaling yaml: %v", err)
	}

	if ver.Variant == "" {
		return nil, report.Report{}, common.ErrNoVariant
	}

	tmp, err := semver.NewVersion(ver.Version)
	if err != nil {
		return nil, report.Report{}, common.ErrInvalidVersion
	}
	version := *tmp

	translator, err := getTranslator(ver.Variant, version)
	if err != nil {
		return nil, report.Report{}, err
	}

	return translator(input, options)
}
