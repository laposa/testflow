<?php

namespace App\Livewire\Reviews;

use App\Models\ReviewRequest;
use Livewire\Component;

class ReviewForm extends Component
{
    public ReviewRequest $reviewRequest;

    public function approve()
    {
        $user = auth()->user();

        $this->reviewRequest->update(['status' => 'approved', 'completed_at' => now()]);
        activity()
            ->event('review_approved')
            ->causedBy($user)
            ->performedOn($this->reviewRequest->reviewable)
            ->log("{$user->name} approved the review.");
        return redirect(request()->header('Referer'));
    }

    public function reject()
    {
        $user = auth()->user();

        $this->reviewRequest->update(['status' => 'rejected', 'completed_at' => now()]);

        activity()
            ->event('review_rejected')
            ->causedBy($user)
            ->performedOn($this->reviewRequest->reviewable)
            ->log("{$user->name} rejected the review.");
        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.review-form');
    }
}
