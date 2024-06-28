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
        $repoName = explode('/', $test['repository_name'])[1];

        return "{$repoName}/{$test['service_name']}/{$test['suite_name']}";
    }
}
