<?php

namespace App\Actions\Github;

use App\Models\Session;
use App\Services\GithubInstallationService;

class DispatchSessionRun
{
    public function handle(Session $session): void
    {
        $client = new GithubInstallationService($session->installation);

        collect($session->items)
            ->groupBy('workflow_id')
            ->map(fn($tests) => $this->dispatchWorkflow($client, $session, $tests->toArray()))
            ->toArray();
    }

    protected function dispatchWorkflow(
        GithubInstallationService $client,
        Session $session,
        array $tests,
    ) {
        $testFilter = collect($tests)
            ->map((fn($test) => explode('.', $test['test_name'])[0]))
            ->join('|');

        $client->dispatchWorkflow(
            repository: $tests[0]['repository_name'],
            workflowId: $tests[0]['workflow_id'],
            inputs: [
                'environment' => $session->environment,
                'test_filter' => $testFilter,
            ],
        );

        // create run in the DB
        // it will be matched with the run ID later (based on timestamp)
        $run = $session->runs()->create([
            'status' => 'unknown',
        ]);

        $run->items()->attach(collect($tests)->pluck('id'));
    }
}
