package cmd

import (
	"fmt"

	"github.com/bytedance/sonic"
	apitypes "github.com/filecoin-project/lotus/api"
	lotusTypes "github.com/filecoin-project/lotus/chain/types"
	parserV1 "github.com/zondax/fil-parser/parser/v1"
	typesV1 "github.com/zondax/fil-parser/parser/v1/types"
	parserV2 "github.com/zondax/fil-parser/parser/v2"
	"github.com/zondax/fil-trace-check/api"
)

func filterTrace(height int64, equivalentAddresses map[string]bool, data []byte) ([]byte, error) {
	switch api.HeightToParserVersion(height) {
	case parserV1.Version:
		return filterTraceV1(equivalentAddresses, data)
	case parserV2.Version:
		return filterTraceV2(equivalentAddresses, data)
	default:
		return nil, fmt.Errorf("unknown compute state version: %s", api.HeightToParserVersion(height))
	}
}

func filterTraceV1(equivalentAddresses map[string]bool, data []byte) ([]byte, error) {
	computeState := &typesV1.ComputeStateOutputV1{}
	err := sonic.Unmarshal(data, &computeState)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling trace: %w", err)
	}

	filteredTraces := []*typesV1.InvocResultV1{}
	for _, trace := range computeState.Trace {
		if trace.MsgRct.ExitCode.IsError() {
			continue
		}
		filteredSubcalls := filterSubcallsV1(equivalentAddresses, trace.ExecutionTrace.Subcalls)
		trace.ExecutionTrace.Subcalls = filteredSubcalls
		added := false
		if trace.Msg != nil {
			if equivalentAddresses[trace.Msg.To.String()] || equivalentAddresses[trace.Msg.From.String()] {
				filteredTraces = append(filteredTraces, trace)
				added = true
			}
		}
		if !added && len(filteredSubcalls) > 0 {
			filteredTraces = append(filteredTraces, trace)
		}
	}
	computeState.Trace = filteredTraces

	res, err := sonic.Marshal(computeState)
	if err != nil {
		return nil, fmt.Errorf("error marshalling filtered trace: %w", err)
	}

	return res, nil
}

func filterSubcallsV1(equivalentAddresses map[string]bool, subcalls []typesV1.ExecutionTraceV1) []typesV1.ExecutionTraceV1 {
	filteredSubcalls := []typesV1.ExecutionTraceV1{}
	nextLevel := []typesV1.ExecutionTraceV1{}

	for _, subcall := range subcalls {
		if subcall.MsgRct != nil && subcall.MsgRct.ExitCode.IsError() {
			continue
		}
		// Collect all nested subcalls for next level processing
		nextLevel = append(nextLevel, subcall.Subcalls...)

		// Check if current subcall matches
		if subcall.Msg != nil {
			if equivalentAddresses[subcall.Msg.To.String()] || equivalentAddresses[subcall.Msg.From.String()] {
				// Create a copy with empty subcalls
				filteredSubcall := subcall
				filteredSubcall.Subcalls = []typesV1.ExecutionTraceV1{}
				filteredSubcalls = append(filteredSubcalls, filteredSubcall)
			}
		}
	}

	// Process next level if there are any subcalls
	if len(nextLevel) > 0 {
		nestedFiltered := filterSubcallsV1(equivalentAddresses, nextLevel)
		filteredSubcalls = append(filteredSubcalls, nestedFiltered...)
	}

	return filteredSubcalls
}

func filterTraceV2(equivalentAddresses map[string]bool, data []byte) ([]byte, error) {
	var computeState apitypes.ComputeStateOutput
	err := sonic.Unmarshal(data, &computeState)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling trace: %w", err)
	}

	filteredTrace := []*apitypes.InvocResult{}
	for _, trace := range computeState.Trace {
		if trace.MsgRct != nil && trace.MsgRct.ExitCode.IsError() {
			continue
		}
		filteredSubcalls := filterSubcallsV2(equivalentAddresses, trace.ExecutionTrace.Subcalls)
		trace.ExecutionTrace.Subcalls = filteredSubcalls
		if trace.Msg != nil {
			if equivalentAddresses[trace.Msg.To.String()] || equivalentAddresses[trace.Msg.From.String()] {
				filteredTrace = append(filteredTrace, trace)
			}
		}
	}
	computeState.Trace = filteredTrace

	res, err := sonic.Marshal(computeState)
	if err != nil {
		return nil, fmt.Errorf("error marshalling filtered trace: %w", err)
	}
	return res, nil
}

func filterSubcallsV2(equivalentAddresses map[string]bool, subcalls []lotusTypes.ExecutionTrace) []lotusTypes.ExecutionTrace {
	filteredSubcalls := []lotusTypes.ExecutionTrace{}
	nextLevel := []lotusTypes.ExecutionTrace{}

	for _, subcall := range subcalls {
		if subcall.MsgRct.ExitCode.IsError() {
			continue
		}
		// Collect all nested subcalls for next level processing
		nextLevel = append(nextLevel, subcall.Subcalls...)

		// Check if current subcall matches
		if equivalentAddresses[subcall.Msg.To.String()] || equivalentAddresses[subcall.Msg.From.String()] {
			// Create a copy with empty subcalls
			filteredSubcall := subcall
			filteredSubcall.Subcalls = []lotusTypes.ExecutionTrace{}
			filteredSubcalls = append(filteredSubcalls, filteredSubcall)
		}
	}

	// Process next level if there are any subcalls
	if len(nextLevel) > 0 {
		nestedFiltered := filterSubcallsV2(equivalentAddresses, nextLevel)
		filteredSubcalls = append(filteredSubcalls, nestedFiltered...)
	}

	return filteredSubcalls
}
