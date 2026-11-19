package parser

import (
	"testing"
)

func TestParseGo(t *testing.T) {
	code := `package auth

import "fmt"

// AuthenticateUser verifies user credentials
func AuthenticateUser(user string, pass string) bool {
	fmt.Println(user)
	srv := &AuthService{}
	return srv.CheckPassword(pass)
}

type AuthService struct {
	Secret string
}

func (s *AuthService) CheckPassword(pass string) bool {
	return pass == s.Secret
}
`
	engine := NewEngine()
	pf, err := engine.Parse("auth.go", []byte(code))
	if err != nil {
		t.Fatalf("unexpected error parsing Go: %v", err)
	}

	if pf.Language != "go" {
		t.Errorf("expected language go, got %s", pf.Language)
	}

	if len(pf.Symbols) != 3 {
		t.Fatalf("expected 3 symbols (function, struct, method), got %d", len(pf.Symbols))
	}

	authFunc := pf.Symbols[0]
	if authFunc.Name != "AuthenticateUser" || authFunc.Kind != KindFunction {
		t.Errorf("unexpected symbol 0: %+v", authFunc)
	}
	if !authFunc.Exported {
		t.Errorf("expected AuthenticateUser to be exported")
	}

	authStruct := pf.Symbols[1]
	if authStruct.Name != "AuthService" || authStruct.Kind != KindStruct {
		t.Errorf("unexpected symbol 1: %+v", authStruct)
	}

	authMethod := pf.Symbols[2]
	if authMethod.Name != "CheckPassword" || authMethod.Kind != KindMethod {
		t.Errorf("unexpected symbol 2: %+v", authMethod)
	}
	if authMethod.Receiver != "AuthService" {
		t.Errorf("expected receiver AuthService, got %s", authMethod.Receiver)
	}

	if len(pf.Calls) == 0 {
		t.Errorf("expected calls to be extracted")
	}
}

func TestParseTypeScript(t *testing.T) {
	code := `import { Logger } from "./logger";

export interface UserSession {
	id: string;
	roles: string[];
}

export class SessionManager {
	private logger: Logger;

	constructor(logger: Logger) {
		this.logger = logger;
	}

	public validate(session: UserSession): boolean {
		this.logger.info("validating session");
		return session.roles.includes("admin");
	}
}

export const createSession = (id: string): UserSession => {
	return { id, roles: ["user"] };
};
`
	engine := NewEngine()
	pf, err := engine.Parse("session.ts", []byte(code))
	if err != nil {
		t.Fatalf("unexpected error parsing TS: %v", err)
	}

	if len(pf.Symbols) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(pf.Symbols))
	}

	names := make(map[string]SymbolKind)
	for _, s := range pf.Symbols {
		names[s.Name] = s.Kind
	}

	if names["UserSession"] != KindInterface {
		t.Errorf("expected UserSession interface, got %v", names["UserSession"])
	}
	if names["SessionManager"] != KindClass {
		t.Errorf("expected SessionManager class, got %v", names["SessionManager"])
	}
	if names["validate"] != KindMethod {
		t.Errorf("expected validate method, got %v", names["validate"])
	}
	if names["createSession"] != KindFunction {
		t.Errorf("expected createSession function, got %v", names["createSession"])
	}
}

func TestParsePython(t *testing.T) {
	code := `import os
from datetime import datetime

class DataPipeline:
	"""Processes raw records from upstream queue."""
	def __init__(self, name: str):
		self.name = name

	def process_record(self, record: dict) -> bool:
		print("record")
		return True

def run_pipeline():
	pipe = DataPipeline("analytics")
	pipe.process_record({})
`
	engine := NewEngine()
	pf, err := engine.Parse("pipeline.py", []byte(code))
	if err != nil {
		t.Fatalf("unexpected error parsing Python: %v", err)
	}

	if len(pf.Symbols) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(pf.Symbols))
	}

	var foundClass, foundMethod, foundFunc bool
	for _, s := range pf.Symbols {
		if s.Name == "DataPipeline" && s.Kind == KindClass {
			foundClass = true
			if s.Docstring != "Processes raw records from upstream queue." {
				t.Errorf("unexpected docstring: %s", s.Docstring)
			}
		}
		if s.Name == "process_record" && s.Kind == KindMethod {
			foundMethod = true
		}
		if s.Name == "run_pipeline" && s.Kind == KindFunction {
			foundFunc = true
		}
	}

	if !foundClass || !foundMethod || !foundFunc {
		t.Errorf("missing expected python symbols: class=%v, method=%v, func=%v", foundClass, foundMethod, foundFunc)
	}
}

func TestParseRust(t *testing.T) {
	code := `use std::collections::HashMap;

pub trait Storage {
	fn get(&self, key: &str) -> Option<String>;
}

pub struct MemoryStorage {
	data: HashMap<String, String>,
}

impl Storage for MemoryStorage {
	fn get(&self, key: &str) -> Option<String> {
		self.data.get(key).cloned()
	}
}
`
	engine := NewEngine()
	pf, err := engine.Parse("storage.rs", []byte(code))
	if err != nil {
		t.Fatalf("unexpected error parsing Rust: %v", err)
	}

	if len(pf.Symbols) < 3 {
		t.Fatalf("expected at least 3 symbols, got %d", len(pf.Symbols))
	}
}

func TestParseJava(t *testing.T) {
	code := `package com.kernel.service;

import java.util.Optional;

public class OrderService {
	public Optional<String> findOrder(String id) {
		System.out.println(id);
		return Optional.of(id);
	}
}
`
	engine := NewEngine()
	pf, err := engine.Parse("OrderService.java", []byte(code))
	if err != nil {
		t.Fatalf("unexpected error parsing Java: %v", err)
	}

	if len(pf.Symbols) < 2 {
		t.Fatalf("expected at least 2 symbols, got %d", len(pf.Symbols))
	}
}
