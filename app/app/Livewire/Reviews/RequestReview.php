<?php

namespace App\Livewire\Reviews;

use App\Enums\SessionActivityType;
use App\Models\Session;
use App\Models\User;
use App\Notifications\ReviewRequestNotification;
use Illuminate\Support\Collection;
use Livewire\Attributes\Validate;
use Livewire\Component;

class RequestReview extends Component
{
    #[Validate('required')]
    public Session $session;

    #[Validate('required')]
    public string $reviewer_id = '';

    /** @var Collection<User> */
    public Collection $users;

    public function mount()
    {
        // Get available reviewers
        $reviewRequests = $this->session->reviewRequests()->where('status', 'pending')->get();

        $this->users = User::whereNotIn('id', $reviewRequests->pluck('reviewer_id'))->get();
    }

    public function save()
    {
        $this->validate();

        $requester = auth()->user();
        $reviewer = User::find($this->reviewer_id);

        $reviewRequest = $this->session->reviewRequests()->create([
            'requester_id' => $requester->id,
            'reviewer_id' => $this->reviewer_id,
        ]);

        $reviewer->notify(new ReviewRequestNotification($reviewRequest));

        $this->session->activity()->create([
            'user_id' => $requester->id,
            'type' => SessionActivityType::review_requested,
            'body' => $requester->name . ' requested a review from ' . $reviewer->name,
        ]);

        session()->flash('status', 'Review requested');

        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.reviews.request-review');
    }
}
