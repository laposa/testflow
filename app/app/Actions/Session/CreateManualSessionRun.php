<?php

namespace App\Actions\Session;

use App\Enums\SessionActivityType;
use App\Models\SessionService;
use App\Models\SessionServiceRun;

class CreateManualSessionRun
{
    public function handle(SessionService $service): SessionServiceRun
    {
        $run = $service->runs()->create([
            'type' => "manual",
            'service_id' => $service->id,
            'status' => 'In Progress', // Change this to fail if any test fails
            'started_at' => now(),
        ]);

        $service->session->activity()->create([
            'user_id' => auth()->id(),
            'type' => SessionActivityType::manual_run_started,
            'body' => "Manual run {$run->id} for {$run->service->displayName} started by " . auth()->user()->name
        ]);

        return $run;
    }
}
