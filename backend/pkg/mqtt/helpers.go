package mqtt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"ws-json-rpc/backend/pkg/generate"
)

// FIXME: Remove regex and use a function. Add tests as well.
var topicParamRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// validateTopicPattern validates an MQTT topic pattern with {param} placeholders.
// Valid patterns:
// - Parameters must be in {paramName} format (e.g., devices/{deviceID}/temperature)
// - Parameter names must start with a letter and contain only alphanumeric characters and underscores
// - Multi-level wildcards '#' are NOT supported for explicitness.
func validateTopicPattern(topic string) error {
	if topic == "" {
		return errors.New("topic cannot be empty")
	}

	segments := strings.Split(topic, "/")

	for i, segment := range segments {
		// Check for multi-level wildcard - not allowed
		if strings.Contains(segment, "#") {
			return errors.New("multi-level wildcard '#' is not supported - use explicit parameters {param} instead")
		}

		// Check for single-level wildcard - should use {param} instead
		if strings.Contains(segment, "+") {
			return errors.New("wildcard '+' is not supported - use parameter syntax {param} instead")
		}

		// Check for parameter syntax
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := segment[1 : len(segment)-1]
			// FIXME: Remove regex and use a function. Add tests as well.
			if !topicParamRegex.MatchString(paramName) {
				return fmt.Errorf("invalid parameter name '%s' - must start with a letter and contain only alphanumeric characters and underscores", paramName)
			}
		} else if strings.Contains(segment, "{") || strings.Contains(segment, "}") {
			return errors.New("invalid parameter syntax - use {paramName} format")
		}

		// Empty segments are only allowed for leading/trailing slashes
		if segment == "" && i != 0 && i != len(segments)-1 {
			return errors.New("empty segments are not allowed in the middle of the topic")
		}
	}

	return nil
}

// convertTopicToMQTT converts a parameterized topic (devices/{deviceID}/temperature)
// to an MQTT wildcard pattern (devices/+/temperature).
func convertTopicToMQTT(topic string) string {
	segments := strings.Split(topic, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			segments[i] = "+"
		}
	}

	return strings.Join(segments, "/")
}

// extractTopicParameters extracts parameter names from a parameterized topic.
// Returns a slice of parameter names in order (e.g., ["deviceID", "sensorType"]).
func extractTopicParameters(topic string) []string {
	var params []string

	segments := strings.SplitSeq(topic, "/")
	for segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			paramName := segment[1 : len(segment)-1]
			params = append(params, paramName)
		}
	}

	return params
}

// validateQoS validates a QoS level.
func validateQoS(qos QoS) error {
	if qos != QoSAtMostOnce && qos != QoSAtLeastOnce && qos != QoSExactlyOnce {
		return errors.New("qos must be 0, 1, or 2")
	}

	return nil
}

func generateParameters(topic string, topicParams []TopicParameter) ([]generate.MQTTTopicParameter, error) {
	var parameters []generate.MQTTTopicParameter
	// Validate path parameters and collect metadata
	params := map[string]struct{}{}
	documentedPathParams := map[string]struct{}{}

	// Extract param names from topic
	for section := range strings.SplitSeq(topic, "/") {
		paramsName, err := generate.ExtractParamName(section)
		if err != nil {
			return nil, fmt.Errorf("invalid topic %s: %w", topic, err)
		}

		for _, paramName := range paramsName {
			params[paramName] = struct{}{}
		}
	}

	// For each documented parameter, validate and collect metadata
	for _, paramSpec := range topicParams {
		if paramSpec.Name == "" {
			return nil, fmt.Errorf("parameter name required for topic %s", topic)
		}

		if paramSpec.Description == "" {
			return nil, fmt.Errorf("parameter Description required for topic %s", topic)
		}

		if paramSpec.Type == nil {
			return nil, fmt.Errorf("parameter Type required for topic %s", topic)
		}

		parameters = append(parameters, generate.MQTTTopicParameter{
			Name:        paramSpec.Name,
			TypeValue:   paramSpec.Type,
			Description: paramSpec.Description,
		})

		if _, exists := params[paramSpec.Name]; !exists {
			return nil, fmt.Errorf("documented path parameter %s not found in path", paramSpec.Name)
		}

		documentedPathParams[paramSpec.Name] = struct{}{}
	}

	// Now go over all discovered path parameters and validate that they are documented
	for name := range params {
		if _, exists := documentedPathParams[name]; !exists {
			return nil, fmt.Errorf("path parameter %s not documented", name)
		}
	}

	return parameters, nil
}
