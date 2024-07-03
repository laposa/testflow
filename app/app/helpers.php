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
        return "{$test['repository_name']}/{$test['service_name']}/{$test['suite_name']}";
    }
}
