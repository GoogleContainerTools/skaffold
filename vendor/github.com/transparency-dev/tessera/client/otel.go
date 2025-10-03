// Copyright 2025 The Tessera authors. All Rights Reserved.
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
// limitations under the License.

package client

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

const name = "github.com/transparency-dev/tessera/client"

var (
	tracer = otel.Tracer(name)
)

var (
	firstKey   = attribute.Key("first")
	NKey       = attribute.Key("N")
	logSizeKey = attribute.Key("logSize")
	indexKey   = attribute.Key("index")
	levelKey   = attribute.Key("level")
	smallerKey = attribute.Key("smaller")
	largerKey  = attribute.Key("larger")
)
