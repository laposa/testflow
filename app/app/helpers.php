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

        //decode to php array
        $xml = simplexml_load_string($run->result_log ?? "");
        $json = json_encode($xml);
        $results = json_decode($json,TRUE);

        //transform to array we build xml from
        if(!empty($results["testsuite"]["testcase"])) {
            //workaround for single test case - returns its own content instead of an array with one test inside
            if($results["testsuite"]["@attributes"]["tests"] == "1") {
                $test = $results["testsuite"]["testcase"];
                $tests[$test["@attributes"]["id"]] = [
                    'test_id' => $test["@attributes"]["id"],
                    'service_id' => $run->service_id,
                    'suite_id' => $run->service->suites->first()->id,
                    'status' => $test["@attributes"]["status"],
                    'comment' => $test["system-out"] ?? "",
                ];
            } else {
                foreach($results["testsuite"]["testcase"] as $test) {
                    $tests[$test["@attributes"]["id"]] = [
                        'test_id' => $test["@attributes"]["id"],
                        'service_id' => $run->service_id,
                        'suite_id' => $run->service->suites->first()->id,
                        'status' => $test["@attributes"]["status"],
                        'comment' => $test["system-out"] ?? "",
                    ];
                }
            }
        }

        return $tests;
    }
}
