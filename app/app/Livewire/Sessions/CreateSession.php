<?php

namespace App\Livewire\Sessions;

use App\Actions\Github\FetchTestSuites;
use Illuminate\Support\Facades\Cache;
use Livewire\Component;

class CreateSession extends Component
{
    public function render(FetchTestSuites $fetchTestSuites)
    {
        $suites = Cache::remember('suites', now()->addHour(), fn() => $fetchTestSuites->handle());

        return view('livewire.sessions.create-session', [
            'suites' => $suites,
        ]);
    }

    public function placeholder()
    {
        return <<<'HTML'
        <div>
            <x-portal-section title="Create new session">
            </x-portal-section>
            <x-loading-overlay></x-loading-overlay>
        </div>
        HTML;
    }
}
