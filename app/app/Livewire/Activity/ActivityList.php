<?php

namespace App\Livewire\Activity;

use App\Models\Session;
use Livewire\Component;
use Livewire\Attributes\On;

class ActivityList extends Component
{
    public Session $session;

    #[On('reload-activity')]
    public function render()
    {
        $activity = $this->session->activity()->orderBy('created_at', 'desc')->get();

        return view('livewire.activity.activity-list', [
            'activity' => $activity,
        ]);
    }
}
