<?php

namespace App\Actions\Github;

use App\Data\ManualTestData;
use App\Models\SessionServiceSuiteTest;
use App\Services\GithubClient;
use App\Services\Installation;
use Illuminate\Support\Arr;
use Symfony\Component\Yaml\Yaml;

class FetchManualTestContent
{
    public function handle(SessionServiceSuiteTest $test): ManualTestData
    {
        $client = new GithubClient();
        $fileContents = $client->fetchRepositoryContents(
            fullName: $test->suite->service->repository_name,
            path: $test->path
        );

        $content = base64_decode($fileContents['content']);
        $parsed = Yaml::parse($content);

        return ManualTestData::from($parsed);
    }
}
