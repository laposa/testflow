import type { TestResult, TestType } from './enums';

export interface Test {
	id: number;
	title: string;
	timestamp: string;
	lastResult: TestResult;
	type: TestType;
	instructions?: string;
	error?: string;
}

export interface Session {
	id: number;
	title: string;
	timestamp: string;
	issuer: string;
	//testSuites: TestSuite[];
	//tests: Test[];
}

export interface TestSuite {
	id: number;
	title: string;
	passed: number;
	failed: number;
	//tests: Test[];
	//remove passed and failed once tests are implemented - should be calculated based on the tests resuluts
}

export interface Run {
	id: number;
	timestamp: string;
	result: TestResult;
	suite?: string;
	test: Test;
	code?: string;
	error?: string;
	session?: string;
}