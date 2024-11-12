@php
    /** @var \App\Models\SessionServiceSuiteTest $test */
@endphp

<div class="manual-test">
    @script
    <script type="text/javascript">

        document.addEventListener('click', function (e) {
            let content = e.target.nextElementSibling;

            if (e.target.classList.contains('expand-input') && !e.target.classList.contains('collapse-input')) {
                content.style.maxHeight = content.scrollHeight + "px";
                e.target.classList.add('collapse-input');
                e.target.innerHTML = 'Hide input';
            } else if (e.target.classList.contains('collapse-input')) {
                content.style.maxHeight = null;
                e.target.classList.remove('collapse-input');
                e.target.innerHTML = 'Show input';
            }
        });

    </script>
    @endscript
    <div class="heading">
        <h2>{{ $test->getInstructions()->description }}</h2>
        <div class="description">
            {{ $test->getInstructions()->suite }}
        </div>
    </div>

    <table>
        <thead>
            <tr>
                <th>Step</th>
                <th>Action</th>
                <th>Expected result</th>
                <th></th>
            </tr>
        </thead>
        <tbody>
            @foreach($test->getInstructions()->steps as $index => $step)
                <tr>
                    <td>{{ $index + 1 }}</td>
                    <td>
                        @foreach($step->action as $action)
                            {{ $action }}
                        @endforeach

                        @if($step->input)
                            <div class="input">
                                <span class="expand-input">Show input</span>
                                <div class="content">
                                    @foreach($step->input as $key => $input)
                                        <!-- TODO copy to clipboard when attribute is added -->
                                        <div><b>{{ $key }}</b>: {{ $input }}</div>
                                    @endforeach
                                </div>
                            </div>
                        @endif
                    </td>
                    <td>
                        <ul class="bullet-list">
                            @foreach($step->result as $result)
                                <li>{{ $result }}</li>
                            @endforeach
                        </ul>
                    </td>
                    <td>
                        <div class="radio-wrapper">
                            <label for="test-step-{{$index}}-pass" class="custom-radio" title="Mark as Passed">
                                <input type="radio" name="test-step-{{$index}}" id="test-step-{{$index}}-pass" />
                                <div class="radio pass"></div>
                                Pass
                            </label>
        
                            <label for="test-step-{{$index}}-fail" class="custom-radio" title="Mark as Failed">
                                <input type="radio" name="test-step-{{$index}}" id="test-step-{{$index}}-fail" />
                                <div class="radio fail"></div>
                                Fail
                            </label>
                        </div>
                    </td>
                </tr>
            @endforeach
        </tbody>
    </table>

    <form wire:submit="save">
        <fieldset>
            <legend>Test Status:</legend>
            <div>
                <input type="radio"
                       id="pass"
                       name="result"
                       value="pass"
                       wire:model="result"
                       required
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
                    required
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
