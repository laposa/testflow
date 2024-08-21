<?php

namespace App\Livewire\Reviews;

use App\Enums\SessionActivityType;
use App\Models\ReviewRequest;
use Livewire\Component;

class ReviewForm extends Component
{
    public ReviewRequest $reviewRequest;

    public function approve()
    {
        $user = auth()->user();

        $this->reviewRequest->update(['status' => 'approved', 'completed_at' => now()]);

        $this->reviewRequest->session->activity()->create([
            'user_id' => $user->id,
            'type' => SessionActivityType::review_approved,
            'body' => $user->name . ' approved the review',
        ]);

        return redirect(request()->header('Referer'));
    }

    public function reject()
    {
        $user = auth()->user();

        $this->reviewRequest->update(['status' => 'rejected', 'completed_at' => now()]);

        $this->reviewRequest->session->activity()->create([
            'user_id' => $user->id,
            'type' => SessionActivityType::review_rejected,
            'body' => $user->name . ' rejected the review',
        ]);

        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.review-form');
    }
}
