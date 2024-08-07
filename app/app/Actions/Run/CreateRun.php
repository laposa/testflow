<?php

namespace App\Actions\Run;

use App\Models\Session;
use Illuminate\Support\Facades\Validator;

class CreateRun
{
    public function handle(Session $session, array $data): void
    {
        $validated = Validator::make($data, [
            'run_id' => ['required', 'string'],
        ])->validate();

        $session->runs()->create([
            'github_run_id' => $validated['run_id'],
        ]);
    }
}
