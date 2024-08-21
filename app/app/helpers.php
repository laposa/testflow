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

if (!function_exists('getTestServiceName')) {
    function getTestServiceName($test)
    {
        $repo = str_replace('laposa/', '', $test['repository_name']);

        return "{$repo}/{$test['service_name']}";
    }
}
