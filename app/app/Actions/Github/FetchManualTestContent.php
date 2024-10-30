<?php

namespace App\Actions\Github;

use App\Data\ManualTestData;
use App\Models\Installation;
use App\Models\SessionServiceSuiteTest;
use App\Services\GithubClient;
use Illuminate\Support\Arr;
use Symfony\Component\Yaml\Yaml;

class FetchManualTestContent
{
    public function handle(SessionServiceSuiteTest $test): ManualTestData
    {
        $installation = Installation::first();
        $client = new GithubClient($installation);
        $fileContents = $client->fetchRepositoryContents(
            fullName: $test->suite->service->repository_name,
            path: $test->path
        );

        $content = base64_decode($fileContents['content']);
        $parsed = Yaml::parse($content);

        return ManualTestData::from($parsed);
    }
}
