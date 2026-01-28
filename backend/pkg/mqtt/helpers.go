package mqtt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var topicParamRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

// ValidateTopicPattern validates an MQTT topic pattern with {param} placeholders.
// Valid patterns:
// - Parameters must be in {paramName} format (e.g., devices/{deviceID}/temperature)
// - Parameter names must start with a letter and contain only alphanumeric characters and underscores
// - Multi-level wildcards '#' are NOT supported for explicitness.
func ValidateTopicPattern(topic string) error {
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

// ConvertTopicToMQTT converts a parameterized topic (devices/{deviceID}/temperature)
// to an MQTT wildcard pattern (devices/+/temperature).
func ConvertTopicToMQTT(topic string) string {
	segments := strings.Split(topic, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			segments[i] = "+"
		}
	}

	return strings.Join(segments, "/")
}

// ExtractTopicParameters extracts parameter names from a parameterized topic.
// Returns a slice of parameter names in order (e.g., ["deviceID", "sensorType"]).
func ExtractTopicParameters(topic string) []string {
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

// ValidateQoS validates a QoS level.
func ValidateQoS(qos QoS) error {
	if qos != QoSAtMostOnce && qos != QoSAtLeastOnce && qos != QoSExactlyOnce {
		return errors.New("qos must be 0, 1, or 2")
	}

	return nil
}
