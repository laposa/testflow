<?php

namespace App\Actions\Session;

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

        foreach ($validated['tests'] as $testJson) {
            $test = json_decode($testJson, true);

            $session->items()->create([
                'repository_id' => $test['repository_id'],
                'workflow_id' => $test['workflow_id'],
                'repository_name' => $test['repository_name'],
                'service_name' => $test['service_name'],
                'suite_name' => $test['suite_name'],
                'test_name' => $test['test_name'],
                'service_url' => $test['service_url'],
            ]);
        }

        return $session;
    }
}
