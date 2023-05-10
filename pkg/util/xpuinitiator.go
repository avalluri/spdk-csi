/*
Copyright (c) Intel Corporation.
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

package util

import (
	"fmt"
	"strings"

	"google.golang.org/grpc"
)

func ParseSpdkXpuTargetType(xpuTargetType string) (string, error) {
	parts := strings.Split(xpuTargetType, "-")
	if parts[0] != "xpu" || len(parts) != 3 {
		return "", fmt.Errorf("invalid xpuTargetType %q", xpuTargetType)
	}
	return parts[1], nil
}

func NewSpdkCsiXpuInitiator(volumeContext map[string]string, xpuConnClient *grpc.ClientConn, xpuTargetType string, kvmPciBridges int) (SpdkCsiInitiator, error) {
	targetType, err := ParseSpdkXpuTargetType(xpuTargetType)
	if err != nil {
		return nil, err
	}
	switch targetType {
	case "sma":
		return NewSpdkCsiSmaInitiator(volumeContext, xpuConnClient, xpuTargetType, kvmPciBridges)
	case "opi":
		return NewSpdkCsiOpiInitiator(volumeContext, xpuConnClient, xpuTargetType, kvmPciBridges)
	default:
		return nil, fmt.Errorf("unknown target type: %q", targetType)
	}
}
