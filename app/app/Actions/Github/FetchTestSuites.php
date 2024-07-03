<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Services\GithubInstallationService;
use Illuminate\Support\Arr;

class FetchTestSuites
{
    protected GithubInstallationService $client;

    public function __construct()
    {
        $installation = Installation::first();
        $this->client = new GithubInstallationService($installation);
    }

    public function handle(): array
    {
        $repositories = $this->client->fetchRepositories();
        $suites = [];

        collect(Arr::get($repositories, 'repositories', []))->each(function ($repository) use (
            &$suites,
        ) {
            $contents = $this->fetchTests($repository);
            if ($contents) {
                $workflows = $this->fetchWorkflows($repository);
                $suites = array_merge(
                    $suites,
                    $this->formatTests($contents, $repository, $workflows),
                );
            }
        });

        return $suites;
    }

    protected function formatTests($contents, $repository, $workflows)
    {
        $tests = [];
        // imitate DB structure for consistency
        foreach ($contents as $service) {
            foreach ($service['children'] as $suite) {
                foreach ($suite['children'] as $test) {
                    $workflow = collect(Arr::get($workflows, 'workflows', []))
                        ->filter(
                            fn($workflow) => str_ends_with(
                                $workflow['path'],
                                getWorkflowFilename($service['name']),
                            ),
                        )
                        ->first();

                    $item = [
                        'repository_id' => $repository['id'],
                        'repository_name' => $repository['name'],
                        'service_name' => $service['name'],
                        'service_url' => $service['url'],
                        'suite_name' => $suite['name'],
                        'test_name' => $test['name'],
                    ];

                    if ($workflow) {
                        $item['workflow_id'] = $workflow['id'];
                    }

                    $tests[] = $item;
                }
            }
        }

        return collect($tests)->groupBy(['repository_name', 'service_name', 'suite_name'])->toArray();
    }

    protected function fetchTests($repository)
    {
        try {
            $latestCommit = $this->client->fetchLatestCommit($repository['full_name']);

            return $this->fetchDirectoryRecursive($repository, [
                'path' => 'tests',
                'url' => "{$repository['html_url']}/tree/{$latestCommit['sha']}/tests",
            ]);
        } catch (\Exception $e) {
            return null;
        }
    }

    protected function fetchDirectoryRecursive($repository, $parent, $level = 1)
    {
        if ($level > 3) {
            return [];
        }

        // the folder structure is tests/{service}/tests/{suite}/{test}
        $path = $parent['path'];
        if ($level == 2) {
            $path = "{$path}/tests";
        }

        $dirs = $this->client->fetchRepositoryContents(
            fullName: $repository['full_name'],
            path: $path,
        );

        if (isset($dirs['status'])) {
            return [];
        }

        // if the directory is a file, return an empty array
        if (isset($dirs['name'])) {
            return [];
        }

        $dirs = collect($dirs)
            ->map(
                fn($dir) => [
                    'full_path' => "{$repository['full_name']}/{$dir['path']}",
                    'path' => $dir['path'],
                    'name' => $dir['name'],
                    'url' => "{$parent['url']}/{$dir['name']}",
                    'children' => $this->fetchDirectoryRecursive($repository, $dir, $level + 1),
                ],
            )
            // filter out everything that starts with an underscore
            // eg. _utils
            ->filter(fn($dir) => $dir['name'][0] !== '_');

        return $dirs->toArray();
    }

    protected function fetchWorkflows($repository)
    {
        try {
            return $this->client->fetchWorkflows($repository['full_name']);
        } catch (\Exception $e) {
            return null;
        }
    }
}
