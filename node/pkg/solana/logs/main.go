package logs

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gagliardetto/solana-go"
)

type (
	LogLineType uint8
	Invocation  struct {
		Program          solana.PublicKey
		Success          bool
		Error            string
		Logs             []LogLine
		ComputeConsumed  uint64
		ComputeAvailable uint64
		Subcalls         []*Invocation
	}

	LogLine struct {
		Type   LogLineType
		Data   [][]byte
		String string
	}
)

const (
	LogLineTypeString LogLineType = 1
	LogLineTypeData   LogLineType = 2
)

var (
	invocationExp  = regexp.MustCompile(`^Program (.*) invoke \[(\d+)]$`)
	programLogExp  = regexp.MustCompile(`^Program log: (.*)$`)
	programDataExp = regexp.MustCompile(`^Program data: (.*)$`)
	consumptionExp = regexp.MustCompile(`^Program (.*) consumed (\d+) of (\d+) compute units$`)
	returnExp      = regexp.MustCompile(`^Program (.*) (success|failed: (.*))$`)
)

func ParseLogs(logLines []string) (invocations []*Invocation, err error) {
	i := 0
	for i < len(logLines) {
		invocation, linesRead, err := parseInvocation(logLines[i:])
		if err != nil {
			return nil, err
		}
		i += linesRead
		invocations = append(invocations, invocation)
	}
	return invocations, err
}

func parseInvocation(logLines []string) (*Invocation, int, error) {
	i := 0
	if len(logLines) < 3 {
		return nil, 0, fmt.Errorf("invocation must have at least 3 lines; has %d", len(logLines))
	}
	invocationMatch := invocationExp.FindStringSubmatch(logLines[0])
	i++
	if invocationMatch == nil {
		return nil, 0, fmt.Errorf("no invocation found in line: %s", logLines[0])
	}
	programKey, err := solana.PublicKeyFromBase58(invocationMatch[1])
	if err != nil {
		return nil, 0, fmt.Errorf("could not parse program key %s: %w", invocationMatch[0], err)
	}

	invocation := &Invocation{
		Program:          programKey,
		Success:          false,
		Logs:             nil,
		ComputeConsumed:  0,
		ComputeAvailable: 0,
		Subcalls:         nil,
	}

	for i < len(logLines) {
		line := logLines[i]
		if logMatch := programLogExp.FindStringSubmatch(line); logMatch != nil {
			invocation.Logs = append(invocation.Logs, LogLine{
				Type:   LogLineTypeString,
				Data:   nil,
				String: logMatch[1],
			})
			i++
			continue
		}
		if dataMatch := programDataExp.FindStringSubmatch(line); dataMatch != nil {
			var data [][]byte

			dataLines := strings.Split(dataMatch[1], " ")
			for _, dataLine := range dataLines {
				lineData, err := base64.StdEncoding.DecodeString(dataLine)
				if err != nil {
					return nil, 0, fmt.Errorf("could not parse data log line %s: %w", dataLine, err)
				}
				data = append(data, lineData)
			}
			invocation.Logs = append(invocation.Logs, LogLine{
				Type:   LogLineTypeData,
				Data:   data,
				String: "",
			})
			i++
			continue
		}
		if consumptionMatch := consumptionExp.FindStringSubmatch(line); consumptionMatch != nil {
			consumptionKey, err := solana.PublicKeyFromBase58(consumptionMatch[1])
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse return program key %s: %w", consumptionMatch[1], err)
			}
			if consumptionKey != programKey {
				return nil, 0, fmt.Errorf("wrong program for consumption update; expected %s; got %s", programKey.String(), consumptionKey.String())
			}
			consumedUnits, err := strconv.Atoi(consumptionMatch[2])
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse consumed units %s: %w", consumptionMatch[2], err)
			}
			invocation.ComputeConsumed = uint64(consumedUnits)
			availableUnits, err := strconv.Atoi(consumptionMatch[3])
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse available units %s: %w", consumptionMatch[3], err)
			}
			invocation.ComputeAvailable = uint64(availableUnits)
			i++
			continue
		}
		if returnMatch := returnExp.FindStringSubmatch(line); returnMatch != nil {
			returnKey, err := solana.PublicKeyFromBase58(returnMatch[1])
			if err != nil {
				return nil, 0, fmt.Errorf("could not parse return program key %s: %w", returnMatch[1], err)
			}
			if returnKey != programKey {
				return nil, 0, fmt.Errorf("wrong program returned; expected %s; got %s", programKey.String(), returnKey.String())
			}
			returnStatus := returnMatch[2]
			switch {
			case returnStatus == "success":
				invocation.Success = true
			case strings.HasPrefix(returnStatus, "failed"):
				invocation.Success = false
				invocation.Error = returnMatch[3]
			default:
				return nil, 0, fmt.Errorf("invalid return status %s", returnStatus)
			}
			i++
			break
		}
		if invocationExp.MatchString(line) {
			subInvocation, linesRead, err := parseInvocation(logLines[i:])
			if err != nil {
				return nil, 0, fmt.Errorf("failed to read subinvocation of %s: %w", programKey.String(), err)
			}
			i += linesRead
			invocation.Subcalls = append(invocation.Subcalls, subInvocation)
			continue
		}

		return nil, 0, fmt.Errorf("unknown log line type: %s", line)
	}

	return invocation, i, nil
}
