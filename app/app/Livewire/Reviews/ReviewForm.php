<?php

namespace App\Livewire\Reviews;

use App\Models\ReviewRequest;
use Livewire\Component;

class ReviewForm extends Component
{
    public ReviewRequest $reviewRequest;

    public function approve()
    {
        $this->reviewRequest->update(['status' => 'approved', 'completed_at' => now()]);
        return redirect(request()->header('Referer'));
    }

    public function reject()
    {
        $this->reviewRequest->update(['status' => 'rejected', 'completed_at' => now()]);
        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.review-form');
    }
}
