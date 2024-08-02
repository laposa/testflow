<?php

namespace App\Livewire\Reviews;

use App\Models\User;
use App\Notifications\ReviewRequestNotification;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Collection;
use Livewire\Attributes\Validate;
use Livewire\Component;

class RequestReview extends Component
{
    #[Validate('required')]
    public Model $model;

    #[Validate('required')]
    public string $reviewer_id = "";

    public $users = [];

    public function mount()
    {
        $this->users = User::all();
    }

    public function save()
    {
        if (!method_exists($this->model, 'reviewRequests')) {
            throw new \InvalidArgumentException("The provided model does not have a 'reviewRequests' relationship.");
        }

        $this->validate();

        $user = User::find($this->reviewer_id);

        $reviewRequest = $this->model->reviewRequests()->create([
            'requester_id' => auth()->id(),
            'reviewer_id' => $this->reviewer_id,
        ]);

        $user->notify(new ReviewRequestNotification($reviewRequest));

        session()->flash("status", "Review requested");

        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.request-review');
    }
}
