<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('test_session_runs', function (Blueprint $table) {
            $table->id();
            $table
                ->foreignId('session_id')
                ->constrained('test_sessions')
                ->onDelete('cascade')
                ->onUpdate('cascade');
            $table->string('status');
            $table->unsignedInteger('passed')->nullable();
            $table->unsignedInteger('failed')->nullable();
            $table->text('result_log')->nullable();
            $table->text('run_log')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('test_session_runs');
    }
};
