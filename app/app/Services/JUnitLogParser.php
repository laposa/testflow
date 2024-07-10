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
        return (int) $this->xmlContent['tests'];
    }

    public function getTotalFailures()
    {
        return (int) $this->xmlContent['failures'];
    }

    public function getTotalPassed()
    {
        return $this->getTotalTests() - $this->getTotalFailures();
    }

    public function getTestCase(string $name)
    {
        $name = strtolower(str_replace('_', ' ', $name));
        $name = str_replace('.js', '', $name);

        foreach ($this->getTestSuites() as $testSuite) {
            foreach ($testSuite['testCases'] as $testCase) {
                if (
                    strtolower($testCase['name']) == $name ||
                    strtolower($testCase['class']) == $name
                ) {
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
            if (strtolower($testSuite['name']) == $name) {
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
                    'status' => 'pass',
                ];

                // remove suite name from the testCase name
                $case['name'] = trim(str_replace($suite['name'], '', $case['name']));

                if ($testCase->failure) {
                    $case['status'] = 'fail';
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
