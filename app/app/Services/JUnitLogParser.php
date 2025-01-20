<?php

namespace App\Services;

class JUnitLogParser
{
    private $xmlContent;

    private $testSuites = [];

    public function __construct($xmlContent)
    {
        $this->xmlContent = simplexml_load_string($xmlContent);
    }

    public function getTotalTests()
    {
        $total = 0;
        foreach ($this->getTestSuites() as $testSuite) {
            $total += $testSuite['tests'];
        }

        return $total;
    }

    public function getTotalFailures()
    {
        $total = 0;
        foreach ($this->getTestSuites() as $testSuite) {
            $total += $testSuite['failures'];
        }

        return $total;
    }

    public function getTotalDuration()
    {
        $total = $this->testSuites['time'] ?? 0;
        foreach ($this->getTestSuites() as $testSuite) {
            $total += $testSuite['time'] ?? 0;
        }

        return $total;
    }

    public function getTotalPassed()
    {
        return $this->getTotalTests() - $this->getTotalFailures();
    }

    public function getTestCase(string $name)
    {
        $name = strtolower(str_replace('_', ' ', $name));
        $name = explode('.', $name)[0];

        foreach ($this->getTestSuites() as $testSuite) {
            foreach ($testSuite['testCases'] as $testCase) {
                $testCaseName = strtolower(str_replace('_', ' ', $testCase['name']));
                $testCaseClass = strtolower(str_replace('_', ' ', $testCase['class']));

                if ($testCaseName == $name || $testCaseClass == $name) {
                    return $testCase;
                }
            }
        }

        return null;
    }

    public function getTestSuite(string $name)
    {
        $name = strtolower(str_replace('_', ' ', $name));

        foreach ($this->getTestSuites() as $testSuite) {
            $testSuiteName = strtolower(str_replace('_', ' ', $testSuite['name']));

            if ($testSuiteName == $name) {
                return $testSuite;
            }
        }

        return null;
    }

    public function getTestSuiteTime(string $name)
    {
        $testSuite = $this->getTestSuite($name);

        return $testSuite ? $testSuite['time'] : 0;
    }

    public function getTestSuites()
    {
        if (!empty($this->testSuites)) {
            return $this->testSuites;
        }

        if (!isset($this->xmlContent->testsuite)) {
            return [];
        }

        $testSuites = [];
        foreach ($this->xmlContent->testsuite as $testSuite) {

            $suite = [
                'name' => (string) $testSuite['name'],
                'tests' => (int) $testSuite['tests'],
                'failures' => (int) $testSuite['failures'],
                'time' => (float) $testSuite['time'],
            ];
            $testCases = [];
            foreach ($testSuite->testcase as $testCase) {

                $case = [
                    'name' => (string) $testCase['name'],
                    'class' => (string) $testCase['classname'],
                    'time' => (float) $testCase['time'],
                    'status' => (string) $testCase['status'],
                    'comment'=> $testCase->{"system-out"}
                ];

                // remove suite name from the testCase name
                $case['name'] = trim(str_replace($suite['name'], '', $case['name']));

                if ($testCase->failure) {
                    $case['failureMessage'] = (string) $testCase->failure;
                }
                $testCases[] = $case;
            }
            $suite['testCases'] = $testCases;
            $testSuites[] = $suite;
        }

        $this->testSuites = $testSuites;

        return $this->testSuites;
    }
}
