import { TestResult } from "../enums";

export function formatTestResult(result: TestResult): string {
	switch(result) {
		case TestResult.Passed:
			return '<span class="pass">Passed</span>';
		case TestResult.Failed:
			return '<span class="fail">Failed</span>';
		case TestResult.NotRun:
			return '<span class="fresh">Not Run</span>';
	}
}

export function formatTimestamp (date: Date): string {
	return date.toLocaleTimeString('en-GB') + ' ' + date.toLocaleDateString('en-GB');
}