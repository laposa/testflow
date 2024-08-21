<?php

namespace App\Livewire\Comments;

use App\Enums\SessionActivityType;
use App\Models\Session;
use Livewire\Attributes\Validate;
use Livewire\Component;

class CreateComment extends Component
{
    #[Validate('required')]
    public Session $session;

    #[Validate('required|min:5')]
    public string $body = '';

    public function save()
    {
        $this->validate();

        $this->session->activity()->create([
            'user_id' => auth()->id(),
            'type' => SessionActivityType::comment,
            ...$this->only('body'),
        ]);

        session()->flash('status', 'Comment created');

        return redirect(request()->header('Referer'));
    }

    public function render()
    {
        return view('livewire.comments.create-comment');
    }
}
