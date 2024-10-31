@php
    /** @var \App\Models\SessionServiceSuiteTest $test */
@endphp

<div>
    {{ $test->getInstructions()->suite }}
    {{ $test->getInstructions()->description }}

    <ol>
        @foreach($test->getInstructions()->steps as $step)
            <li>
                <div>
                    <strong>Action:</strong>
                    @foreach($step->action as  $action)
                        <div>{{ $action }}</div>
                    @endforeach
                </div>
                @if (count($step->input) > 0)
                    <div>
                        <strong>Input:</strong>
                        @foreach($step->input as $key => $input)
                            <div>{{ $key }}: {{ $input }}</div>
                        @endforeach
                    </div>
                @endif
                <div>
                    <strong>Result:</strong>
                    <ul>
                        @foreach($step->result as $result)
                            <li>
                                {{ $result }}
                            </li>
                        @endforeach
                    </ul>
                </div>
            </li>
        @endforeach
    </ol>
    <form wire:submit="save">
        <fieldset>
            <legend>Test Status:</legend>
            <div>
                <input type="radio"
                       id="pass"
                       name="result"
                       value="pass"
                       wire:model="result"
                />
                <label for="pass">Pass</label>
            </div>

            <div>
                <input
                    type="radio"
                    id="fail"
                    value="fail"
                    name="result"
                    wire:model="result"
                />
                <label for="fail">Fail</label>
            </div>
        </fieldset>

        <div>
            <label for="comment">Comment</label>
            <textarea
                id="comment"
                name="comment"
                wire:model="comment"></textarea>
        </div>


        <button type="submit">Submit</button>
    </form>
</div>
