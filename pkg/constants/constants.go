package constants

const (
	// ConfigFilename is the installer configuration file (framework contract).
	// All installers using this framework must provide this file.
	ConfigFilename = "config.yaml"

	// ValuesFilename is the values template file (framework contract).
	// All installers using this framework must provide this file.
	ValuesFilename = "values.yaml.tpl"

	// InstructionsFilename is the MCP instructions file (framework convention).
	// This file provides instructions for the Model Context Protocol server.
	InstructionsFilename = "instructions.md"
)
