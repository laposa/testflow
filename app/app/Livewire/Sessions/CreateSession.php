<?php

namespace App\Livewire\Sessions;

use Livewire\Component;
use Illuminate\Support\Facades\Cache;
use App\Actions\Github\FetchTestSuites;

class CreateSession extends Component
{
    public function render(FetchTestSuites $fetchTestSuites)
    {
        $suites = Cache::remember('suites', now()->addHour(), fn () => $fetchTestSuites->handle());

        $suitesWithoutWorkflow = collect($suites)->filter(fn ($suite) => !isset($suite[0]['workflow_id']))->toArray();
        $suites = collect($suites)->filter(fn ($suite) => isset($suite[0]['workflow_id']))->toArray();

        return view('livewire.sessions.create-session', [
            'suites' => $suites,
            'suitesWithoutWorkflow' => $suitesWithoutWorkflow,
        ]);
    }

    public function placeholder()
    {
        return <<<'HTML'
        <div>
            <x-portal-section title="Create new session" width="full">
            </x-portal-section>
            <x-loading-overlay></x-loading-overlay>
        </div>
        HTML;
    }
}
