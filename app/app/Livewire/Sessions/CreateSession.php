<?php

namespace App\Livewire\Sessions;

use App\Actions\Github\FetchTests;
use Illuminate\Support\Facades\Cache;
use Livewire\Component;

class CreateSession extends Component
{
    public function render(FetchTests $fetchTests)
    {
        $tests = Cache::remember('tests', now()->addHour(), fn() => $fetchTests->handle());

        return view('livewire.sessions.create-session', [
            'tests' => $tests,
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
