package routing

import (
	"fmt"
	"strings"

	"github.com/SamCullin/gh-router/internal/config"
	"github.com/SamCullin/gh-router/internal/target"
)

type Source string

const (
	SourceOverride    Source = "override"
	SourceRepo        Source = "repo"
	SourceOrg         Source = "org"
	SourcePath        Source = "path"
	SourceImplicitOrg Source = "implicit-org"
	SourceDefault     Source = "default"
)

type AccountResolution struct {
	Account string
	Source  Source
	Rule    string
}

func ResolveAccount(configuration config.Config, resolvedTarget target.Target, override string) (AccountResolution, error) {
	if strings.TrimSpace(override) != "" {
		return AccountResolution{Account: configuration.AccountName(strings.TrimSpace(override)), Source: SourceOverride}, nil
	}
	if resolvedTarget.Repository != "" {
		if account, rule, ok := configuration.RepositoryRule(resolvedTarget.Repository); ok {
			return AccountResolution{Account: configuration.AccountName(account), Source: SourceRepo, Rule: rule}, nil
		}
	}
	organisation := resolvedTarget.Organisation
	if organisation == "" && resolvedTarget.Repository != "" {
		organisation = strings.SplitN(resolvedTarget.Repository, "/", 2)[0]
	}
	if organisation != "" {
		if account, rule, ok := configuration.OrganisationRule(organisation); ok {
			return AccountResolution{Account: configuration.AccountName(account), Source: SourceOrg, Rule: rule}, nil
		}
	}
	if resolvedTarget.Directory != "" {
		if account, rule, ok := configuration.PathRule(resolvedTarget.Directory); ok {
			return AccountResolution{Account: configuration.AccountName(account), Source: SourcePath, Rule: rule}, nil
		}
	}
	if organisation != "" {
		if rule, configuredOrganisation, ok := configuration.ImplicitOrganisationRule(organisation); ok {
			return AccountResolution{Account: configuration.AccountName(rule.Account), Source: SourceImplicitOrg, Rule: configuredOrganisation}, nil
		}
	}
	if strings.TrimSpace(configuration.Default) == "" {
		return AccountResolution{}, fmt.Errorf("default account is not configured")
	}
	return AccountResolution{Account: configuration.AccountName(configuration.Default), Source: SourceDefault}, nil
}
