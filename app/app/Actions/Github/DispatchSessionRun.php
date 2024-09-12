<?php

namespace App\Actions\Github;

use App\Enums\SessionActivityType;
use App\Models\Session;
use App\Models\SessionService;
use App\Services\GithubInstallationService;

class DispatchSessionRun
{
    public function handle(Session $session, ?array $servicesIds): void
    {
        $client = new GithubInstallationService($session->installation);

        $services = $session->services()->with('suites.tests');
        if ($servicesIds && count($servicesIds) > 0) {
            $services = $services->whereIn('id', $servicesIds);
        }

        $services = $services->get();
        $runs = [];

        foreach ($services as $service) {
            $run = $this->dispatchWorkflow($client, $session, $service);
            $runs[] = ['run' => $run, 'service' => $service];
        }

        $body = auth()->user()->name . ' has executed: ';
        foreach ($runs as $i => $run) {
            if ($i > 0) {
                $body .= ', ';
            }

            $body .= $run['service']->displayName . ' with run ID ' . $run['run']->id;
        }

        $session->activity()->create([
            'user_id' => auth()->id(),
            'type' => SessionActivityType::run_dispatched,
            'body' => $body,
        ]);
    }

    protected function dispatchWorkflow(
        GithubInstallationService $client,
        Session $session,
        SessionService $service,
    ) {
        $tests = $service->suites->flatMap(function ($suite) {
            return $suite->tests->map(function ($test) use ($suite) {
                $test['suite'] = $suite;
                return $test;
            });
        });

        if ($service->name === 'mobile') {
            $testFilter = $tests->map(fn($test) => $test['suite']->name)->unique()->join(',');
        } elseif ($service->name === 'web') {
            $testFilter = collect($tests)
                ->map(fn($item) => "tests/{$item['suite']->name}/{$item['name']}")
                ->join(',');
        } elseif ($service->name === 'api') {
            $testFilter = collect($tests)
                ->map(fn($test) => explode('.', \Str::replace('_', ' ', $test['name']))[0])
                ->join('|');
        }

        $client->dispatchWorkflow(
            repository: $service->repository_name,
            workflowId: $service->workflow_id,
            inputs: [
                'environment' => $session->environment,
                'test_filter' => $testFilter,
            ],
        );

        // create run in the DB
        // it will be matched with the run ID later (based on timestamp)
        $run = $service->runs()->create([
            'status' => 'unknown',
        ]);

        return $run;
    }
}
