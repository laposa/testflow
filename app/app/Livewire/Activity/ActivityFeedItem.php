<?php

namespace App\Livewire\Activity;

use App\Models\Comment;
use App\Models\ReviewRequest;
use Livewire\Component;
use Spatie\Activitylog\Models\Activity;

class ActivityFeedItem extends Component
{
    public Comment|Activity $entry;

    public function delete()
    {
        $this->entry->delete();
        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.activity.activity-feed-item');
    }
}
