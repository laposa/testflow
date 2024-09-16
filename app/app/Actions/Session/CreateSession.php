<?php

namespace App\Actions\Session;

use App\Enums\SessionActivityType;
use App\Models\Installation;
use Illuminate\Support\Facades\Validator;

class CreateSession
{
    public function handle(Installation $installation, array $data)
    {
        $validated = Validator::make(
            $data,
            [
                'name' => ['required', 'string'],
                'environment' => ['required', 'string'],
                'tests' => ['required', 'array', 'min:1'],
                'tests.*' => ['required', 'json'],
            ],
            [
                'tests' => 'Please select at least one test suite.',
            ],
        )->validate();

        $session = $installation->sessions()->create([
            'name' => $validated['name'],
            'environment' => $validated['environment'],
            'issuer_id' => auth()->id(),
        ]);
        $session->save();

        $services = [];
        $suites = [];

        foreach ($validated['tests'] as $testJson) {
            $test = json_decode($testJson, true);

            $serviceId = $test['repository_name'] . $test['service_name'];
            if (!isset($services[$serviceId])) {
                $service = $session->services()->create([
                    'repository_id' => $test['repository_id'],
                    'workflow_id' => $test['workflow_id'],
                    'name' => $test['service_name'],
                    'url' => $test['service_url'],
                    'repository_name' => $test['repository_name'],
                ]);
                $service->save();

                $services[$serviceId] = $service;
            }

            $suiteId = $serviceId . $test['suite_name'];
            if (!isset($suites[$suiteId])) {
                $suite = $services[$serviceId]->suites()->create([
                    'name' => $test['suite_name'],
                    'url' => $test['service_url'] . '/' . $test['suite_name'],
                ]);
                $suite->save();

                $suites[$suiteId] = $suite;
            }

            $suites[$suiteId]->tests()->create([
                'name' => $test['test_name'],
                'url' =>
                    $test['service_url'] . '/' . $test['suite_name'] . '/' . $test['test_name'],
            ]);
        }

        $session->activity()->create([
            'user_id' => auth()->id(),
            'type' => SessionActivityType::session_created,
            'body' => auth()->user()->name . ' created the session.',
        ]);

        return $session;
    }
}
