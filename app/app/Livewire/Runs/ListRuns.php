<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Models\Session;
use Livewire\Component;

class ListRuns extends Component
{
    public Session $session;

    public bool $isLoading = false;

    public array $selectedServices = [];
    public array $displayedRuns = [];

    public function mounted(Session $session)
    {
        $this->session = $session;
    }

    public function render(FetchSessionWorkflowRuns $fetchSessionWorkflowRuns)
    {
        $this->session->load([
            'services.runs' => function ($query) {
                $query->orderBy('created_at', 'desc');
            },
        ]);

        $updated = $fetchSessionWorkflowRuns->handle($this->session);

        if ($updated) {
            $this->session->load([
                'services.runs' => function ($query) {
                    $query->orderBy('created_at', 'desc');
                },
            ]);
        }

        return view('livewire.runs.list-runs', [
            'session' => $this->session,
            'pollingEnabled' => true,
        ]);
    }

    public function createRun(DispatchSessionRun $dispatchSessionRun)
    {
        $dispatchSessionRun->handle($this->session, $this->selectedServices);
        $this->dispatch('reload-activity');
    }
}
