<?php

namespace App\Livewire\Reviews;

use App\Models\Comment;
use App\Models\ReviewRequest;
use Illuminate\Database\Eloquent\Model;
use Livewire\Component;
use Spatie\Activitylog\Models\Activity;

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
