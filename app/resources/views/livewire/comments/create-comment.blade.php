<div>
    <form wire:submit="save" class="create-comment">
        @csrf
        <div class="form-group">
            <label for="body">Add a comment</label>
            <textarea id="body" wire:model="body"></textarea>
            @error('body')
                <span class="error">{{ $message }}</span>
            @enderror
        </div>
        <button class="button filled" type="submit">Save</button>
    </form>
</div>
