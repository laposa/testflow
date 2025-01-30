<?php

if (!function_exists('getWorkflowFilename')) {
    function getWorkflowFilename(string $serviceName)
    {
        return "run-tests-$serviceName.yaml";
    }
}

if (!function_exists('getTestSuiteName')) {
    function getTestSuiteName($test)
    {
        $repo = str_replace('laposa/', '', $test['repository_name']);

        return "{$repo}/{$test['service_name']}/{$test['suite_name']}";
    }
}

if (!function_exists('getResultsFromXML')) {
    function getResultsFromXML($run)
    {
        $tests = [];

        $xml = simplexml_load_string($run->result_log ?? "");
        if (!$xml || $run->result_log == "") {
            return $tests;
        }

        foreach ($xml->testsuite as $suite) {
            foreach ($suite->testcase as $test) {
                $tests[(int) $test['id']] = [
                    'test_id' => (int) $test['id'],
                    'service_id' => $run->service_id,
                    'suite_id' => $run->service->suites->first()->id,
                    'status' => (string) $test['status'],
                    'comment' => (string) $test->{'system-out'} ?? "",
                ];
            }
        }

        return $tests;
    }
}

if (!function_exists('validateManualTest')) {
    function validateManualTest($test) 
    {
        $error = "";

        //check if name corresponds with filename
        $name = $test->getInstructions()->name . ".manual.yaml";
        if($name != $test->name) {
            $error .= "<li>Filename does not match the name of the file</li>";
        }
        
        return $error;
    }
}
