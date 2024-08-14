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
    public string $reviewer_id = '';

    /** @var Collection<User> */
    public Collection $users;

    public function mount()
    {
        if (!method_exists($this->model, 'reviewRequests')) {
            throw new \InvalidArgumentException(
                "The provided model does not have a 'reviewRequests' relationship.",
            );
        }

        // Get available reviewers
        $reviewRequests = $this->model->reviewRequests()->where('status', 'pending')->get();

        $this->users = User::whereNotIn('id', $reviewRequests->pluck('reviewer_id'))->get();
    }

    public function save()
    {
        if (!method_exists($this->model, 'reviewRequests')) {
            throw new \InvalidArgumentException(
                "The provided model does not have a 'reviewRequests' relationship.",
            );
        }

        $this->validate();

        $requester = auth()->user();
        $reviewer = User::find($this->reviewer_id);

        $reviewRequest = $this->model->reviewRequests()->create([
            'requester_id' => auth()->id(),
            'reviewer_id' => $this->reviewer_id,
        ]);

        $reviewer->notify(new ReviewRequestNotification($reviewRequest));

        activity()
            ->causedBy($requester)
            ->event('review_request')
            ->performedOn($this->model)
            ->log("{$requester->name} requested a review from {$reviewer->name}");

        session()->flash('status', 'Review requested');

        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.request-review');
    }
}
