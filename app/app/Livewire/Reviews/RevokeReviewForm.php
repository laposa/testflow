<?php

namespace App\Livewire\Reviews;

use App\Models\ReviewRequest;
use Livewire\Component;

class RevokeReviewForm extends Component
{
    public ReviewRequest $reviewRequest;

    public function delete()
    {
        $this->reviewRequest->delete();
        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.revoke-review-form');
    }
}
