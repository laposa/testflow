<?php

namespace App\Actions\Session;

use App\Enums\SessionActivityType;
use App\Models\Session;
use Illuminate\Support\Facades\Validator;

class CreateSession
{
    protected array $services = [];
    protected array $suites = [];
    public function handle(array $data)
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

        $session = Session::create([
            'name' => $validated['name'],
            'environment' => $validated['environment'],
            'issuer_id' => auth()->id(),
        ]);
        $session->save();

        collect($validated['tests'])
            ->map(fn ($test) => json_decode($test, true))
            ->groupBy(fn ($test) => \Str::contains( $test['test_name'], 'manual') ? 'manual' : 'automated')
            ->map(fn($tests, $type) => $this->createService($session, $type, $tests->toArray()));

            $session->activity()->create([
                'user_id' => auth()->id(),
                'type' => SessionActivityType::session_created,
                'body' => auth()->user()->name . ' created the session.',
            ]);

        return $session;
    }

    protected function createService(Session $session, string $type, array $tests)
    {
        collect($tests)
            ->groupBy(fn ($test) => $test['repository_name'] . $test['service_name'] . " ($type)")
            ->each(function($tests) use ($session, $type) {
                $service = $session->services()->create([
                    'repository_id' => $tests[0]['repository_id'],
                    'workflow_id' => $tests[0]['workflow_id'],
                    'name' => $tests[0]['service_name'],
                    'type' => $type,
                    'path' => $tests[0]['service_path'],
                    'repository_name' => $tests[0]['repository_name'],
                    'commit_sha' => $tests[0]['commit_sha'],
                    'branch' => 'master',
                ]);

                $tests
                    ->groupBy(fn ($test) => $test['suite_name'])
                    ->each(function($tests) use ($service) {
                        $suite = $service->suites()->create([
                            'name' => $tests[0]['suite_name'],
                            'path' => $tests[0]['suite_path'],
                        ]);

                        $suite->tests()->createMany(
                            $tests->map(fn ($test) => [
                                'name' => $test['test_name'],
                                'path' => $test['test_path'],
                            ])->toArray()
                        );
                    });
            });
    }
}
