<?php

namespace App\Livewire\Runs;

use App\Actions\Github\DispatchSessionRun;
use App\Actions\Github\FetchSessionWorkflowRuns;
use App\Actions\Session\CreateManualSessionRun;
use App\Enums\SessionActivityType;
use App\Models\Session;
use App\Models\SessionService;
use Livewire\Attributes\Computed;
use Livewire\Attributes\On;
use Livewire\Component;

class ListRuns extends Component
{
    public Session $session;
    public bool $pollingEnabled = true;
    public array $selectedServices = [];

    #[On('list-runs.refresh')]
    public function refresh() {}

    /**
     * Computed property to get the selected services that have manual tests
     */
    #[Computed]
    public function selectedManualServices()
    {
        return collect($this->selectedServices)->map(fn ($id) => $this->session->services->find($id))->filter(fn ($service) => $service->has_manual_tests);
    }

    /**
     * Refresh the session workflow runs from github
     * Cache the response to speed up render time
     */
    public function refreshSessionWorkflowRuns(bool $cached = true)
    {
        $fetchSessionWorkflowRuns = app(FetchSessionWorkflowRuns::class);

        if ($cached) {
            $updated = \Cache::remember(
                "fetch-session-workflow-runs-{$this->session->id}",
                now()->addSeconds(120),
                fn() => $fetchSessionWorkflowRuns->handle($this->session)
            );
        } else {
            $updated = $fetchSessionWorkflowRuns->handle($this->session);
        }

        if ($updated) {
            $this->session->load([
                'services.runs' => function ($query) {
                    $query->orderBy('created_at', 'desc');
                },
            ]);
        }
    }

    public function createRun()
    {
        $this->session->services()
            ->where(function($query) {
                if (count($this->selectedServices) > 0) {
                    $query->whereIn('id', $this->selectedServices);
                }
            })
            ->get()
            ->each(function (SessionService $service) {
                if ($service->type === 'automated') {
                    (new DispatchSessionRun())->handle($this->session, [$service->id]);
                } else if ( $service->type === 'manual') {
                    (new CreateManualSessionRun())->handle($service);
                }
            });
    }


    public function render()
    {
        $this->session->load([
            'services.runs' => function ($query) {
                $query->orderBy('created_at', 'desc');
            },
        ]);

        $this->refreshSessionWorkflowRuns();

        return view('livewire.runs.list-runs');
    }

}
