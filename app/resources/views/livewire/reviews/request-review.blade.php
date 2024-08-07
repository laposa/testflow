@php
    /** @var \Illuminate\Support\Collection|\App\Models\User[] $users */
    /** @var string $reviewer_id */
@endphp
<div>
    <form wire:submit="save" class="request-review">
        @csrf

        <div class="form-group">
            <select name="reviewer_id" wire:model="reviewer_id">
                <option value="">Select a reviewer</option>
                @foreach ($users as $user)
                    <option value="{{$user->id}}">{{$user->name}}</option>
                @endforeach
            </select>
            @error('reviewer_id') <span class="error">{{ $message }}</span> @enderror
        </div>

        <button class="button filled" type="submit">Request</button>
    </form>
</div>
