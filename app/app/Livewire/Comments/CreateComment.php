<?php

namespace App\Livewire\Comments;

use App\Models\Session;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Facades\Validator;
use Livewire\Attributes\Validate;
use Livewire\Component;

class CreateComment extends Component
{
    #[Validate('required')]
    public Model $model;

    #[Validate('required|min:5')]
    public string $body = '';

    public function save()
    {
        // check $model is commentable
        if (!method_exists($this->model, 'comments')) {
            throw new \InvalidArgumentException(
                "The provided model does not have a 'comments' relationship.",
            );
        }

        $this->validate();

        $this->model->comments()->create([
            'user_id' => auth()->id(),
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
