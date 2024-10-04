<?php

namespace App\Actions\Github;

use App\Models\Installation;
use App\Services\GithubClient;
use Illuminate\Support\Arr;

class FetchTests
{
    protected GithubClient $client;

    public function __construct()
    {
        $installation = Installation::first();
        $this->client = new GithubClient($installation);
    }

    public function handle(): array
    {
        $repositories = $this->client->fetchRepositories();
        $tests = [];

        collect(Arr::get($repositories, 'repositories', []))->each(function ($repository) use (
            &$tests,
        ) {
            $contents = $this->fetchTests($repository);
            if ($contents) {
                $workflows = $this->fetchWorkflows($repository);
                $tests = array_merge(
                    $tests,
                    $this->formatTests($contents, $repository, $workflows),
                );
            }
        });

        return $tests;
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
                        'repository_name' => $repository['full_name'],
                        'commit_sha' => $service['commit_sha'],
                        'service_name' => $service['name'],
                        'service_path' => $service['path'],
                        'suite_name' => $suite['name'],
                        'suite_path' => $suite['path'],
                        'test_name' => $test['name'],
                        'test_path' => $test['path'],
                    ];

                    if ($workflow) {
                        $item['workflow_id'] = $workflow['id'];
                    }

                    $tests[] = $item;
                }
            }
        }

        return collect($tests)
            ->groupBy(['repository_name', 'service_name', 'suite_name'])
            ->toArray();
    }

    protected function fetchTests($repository)
    {
        try {
            $latestCommit = $this->client->fetchLatestCommit($repository['full_name']);

            return $this->fetchDirectoryRecursive($repository, [
                'path' => 'tests',
                'url' => "{$repository['html_url']}/tree/{$latestCommit['sha']}/tests",
                'commit_sha' => $latestCommit['sha'],
            ]);
        } catch (\Exception $e) {
            dd($e);
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
                    'commit_sha' => isset($parent['commit_sha']) ? $parent['commit_sha'] : null,
                    'full_path' => "{$services['full_name']}/{$dir['path']}",
                    'path' => $dir['path'],
                    'name' => $dir['name'],
                    'url' => "{$parent['url']}/{$dir['name']}",
                    'children' => $this->fetchDirectoryRecursive($services, $dir, $level + 1),
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
